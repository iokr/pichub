package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/pichub/pichub/internal/auth"
	"github.com/pichub/pichub/internal/cache"
	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/cron"
	"github.com/pichub/pichub/internal/crypto"
	"github.com/pichub/pichub/internal/handler"
	"github.com/pichub/pichub/internal/imageproc"
	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/internal/mail"
	"github.com/pichub/pichub/internal/middleware"
	"github.com/pichub/pichub/internal/queue"
	"github.com/pichub/pichub/internal/repository"
	"github.com/pichub/pichub/internal/router"
	"github.com/pichub/pichub/internal/s3gw"
	"github.com/pichub/pichub/internal/service"
	"github.com/pichub/pichub/internal/storage/manager"
	"github.com/pichub/pichub/internal/web"
)

// App 持有进程级资源与生命周期。
type App struct {
	cfg *config.Config
	log *logger.Logger

	db       *gorm.DB
	rdb      *redis.Client
	httpSrv  *http.Server
	worker   *queue.Worker
	cron     *cron.Scheduler
	buffer   *cache.RefererBuffer
	services *service.Services

	ctx    context.Context
	cancel context.CancelFunc

	startOnce sync.Once
	stopOnce  sync.Once
}

// New 按 DB → Redis → Repo → Service → Handler → Router 的顺序装配依赖。
func New(cfg *config.Config, log *logger.Logger) (*App, error) {
	db, err := NewDB(cfg, log)
	if err != nil {
		return nil, err
	}
	if cfg.Migrate.OnStart {
		if err := RunMigrate(db, log); err != nil {
			closeDB(db)
			return nil, err
		}
	}
	rdb, err := NewRedis(cfg, log)
	if err != nil {
		closeDB(db)
		return nil, err
	}

	repos := repository.New(db)
	cipher, err := crypto.New(cfg.Security.SecretKey)
	if err != nil {
		closeAll(db, rdb)
		return nil, fmt.Errorf("init cipher: %w", err)
	}
	jwtMgr := auth.NewManager(&cfg.Security)
	blacklist := cache.NewTokenBlacklist(rdb, cfg.Redis.KeyPrefix)
	limiter := cache.NewRateLimiter(rdb, cfg.Redis.KeyPrefix)
	captcha := cache.NewCaptcha(rdb, cfg.Captcha, cfg.Redis.KeyPrefix)
	deduper := cache.NewDeduper(rdb, cfg.Redis.KeyPrefix)
	buffer := cache.NewRefererBuffer(rdb, cfg.Redis.KeyPrefix, cfg.Referer.BufferSize, log)
	mailer := mail.New(cfg.Mail, log)
	processor := imageproc.New(cfg.Image, log)

	storages, err := manager.NewManager(cfg.Storage, repos.Storage, cipher, log)
	if err != nil {
		closeAll(db, rdb)
		return nil, fmt.Errorf("init storage manager: %w", err)
	}
	initCtx, initCancel := context.WithTimeout(context.Background(), 15*time.Second)
	err = storages.EnsureDefault(initCtx)
	initCancel()
	if err != nil {
		closeAll(db, rdb)
		return nil, fmt.Errorf("ensure default storage: %w", err)
	}

	q := queue.NewRedisQueue(rdb, cfg.Redis.KeyPrefix, log)
	services := service.New(service.Deps{
		Config:     cfg,
		Log:        log,
		Repos:      serviceRepos(repos),
		JWT:        jwtMgr,
		Blacklist:  blacklist,
		Limiter:    limiter,
		Captcha:    captcha,
		Deduper:    deduper,
		Buffer:     buffer,
		Cipher:     cipher,
		Mailer:     mailer,
		Queue:      q,
		QueueStats: q,
		Storages:   storages,
		Processor:  processor,
	})

	registry := queue.NewRegistry()
	registry.Register(queue.TaskSendEmail, func(ctx context.Context, task *queue.Task) error {
		var payload queue.SendEmailPayload
		if err := json.Unmarshal(task.Payload, &payload); err != nil {
			return fmt.Errorf("decode email payload: %w", err)
		}
		return services.Email.HandleTask(ctx, payload)
	})
	worker := queue.NewWorker(q, registry, cfg.Queue, log)

	scheduler := cron.New(cfg, cron.Deps{
		Images:    services.Image,
		Referer:   services.Referer,
		Upload:    services.Upload,
		Processor: processor,
		Emails:    repos.Email,
		Logs:      repos.Log,
	}, log)

	ready := func(ctx context.Context) error {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			return fmt.Errorf("database: %w", err)
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis: %w", err)
		}
		return nil
	}

	h := handler.New(cfg, log, services, ready)
	authn := middleware.NewAuthenticator(log, jwtMgr, blacklist, services.Token, services.User)
	lim := middleware.NewLimiter(limiter, log)
	gateway := s3gw.New(cfg, log, services, deduper)
	webFS, _ := web.FS()
	engine := router.New(router.Options{
		Config:  cfg,
		Log:     log,
		Handler: h,
		Auth:    authn,
		Limiter: lim,
		S3:      gateway,
		Web:     webFS,
	})

	srv := &http.Server{
		Addr:           cfg.Server.Addr,
		Handler:        engine,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	rootCtx, cancel := context.WithCancel(context.Background())
	return &App{
		cfg:      cfg,
		log:      log,
		db:       db,
		rdb:      rdb,
		httpSrv:  srv,
		worker:   worker,
		cron:     scheduler,
		buffer:   buffer,
		services: services,
		ctx:      rootCtx,
		cancel:   cancel,
	}, nil
}

// Run 启动后台任务并阻塞运行 HTTP 服务。
func (a *App) Run() error {
	a.startOnce.Do(func() {
		if a.buffer != nil {
			a.buffer.Start(a.ctx)
		}
		if a.worker != nil {
			a.worker.Start(a.ctx)
		}
		if a.cron != nil {
			a.cron.Start(a.ctx)
		}
	})
	if a.log != nil {
		a.log.Info("http server listening", logger.String("addr", a.cfg.Server.Addr))
	}
	err := a.httpSrv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown 优雅关闭 HTTP、后台任务与外部连接。
func (a *App) Shutdown(ctx context.Context) error {
	var err error
	a.stopOnce.Do(func() {
		if a.log != nil {
			a.log.Info("shutting down application")
		}
		if a.httpSrv != nil {
			if serr := a.httpSrv.Shutdown(ctx); serr != nil {
				err = serr
			}
		}
		if a.cancel != nil {
			a.cancel()
		}
		if a.worker != nil {
			a.worker.Stop()
		}
		if a.cron != nil {
			a.cron.Stop()
		}
		if a.buffer != nil {
			a.buffer.Stop()
		}
		if a.rdb != nil {
			_ = a.rdb.Close()
		}
		closeDB(a.db)
	})
	return err
}

// serviceRepos 把具体的仓储实现适配为业务层接口集合。
func serviceRepos(r *repository.Repos) service.Repos {
	return service.Repos{
		Tx:      r.Tx,
		User:    r.User,
		Group:   r.Group,
		Image:   r.Image,
		Album:   r.Album,
		Tag:     r.Tag,
		Token:   r.Token,
		S3Cred:  r.S3Cred,
		Share:   r.Share,
		Referer: r.Referer,
		Storage: r.Storage,
		Email:   r.Email,
		Setting: r.Setting,
		Invite:  r.Invite,
		Log:     r.Log,
		Counter: r.Counter,
	}
}

func closeDB(db *gorm.DB) {
	if db == nil {
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func closeAll(db *gorm.DB, rdb *redis.Client) {
	if rdb != nil {
		_ = rdb.Close()
	}
	closeDB(db)
}
