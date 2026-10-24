<script setup lang="ts">
import { computed } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, LineChart, PieChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components'

use([CanvasRenderer, LineChart, BarChart, PieChart, GridComponent, TooltipComponent, LegendComponent])

const props = withDefaults(
  defineProps<{
    data: { date: string; hits: number }[]
    height?: string
    name?: string
  }>(),
  {
    height: '280px',
    name: '引用次数',
  },
)

const option = computed(() => ({
  grid: { left: 48, right: 20, top: 24, bottom: 32 },
  tooltip: { trigger: 'axis' },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: props.data.map((item) => item.date),
    axisLine: { lineStyle: { color: '#E5E6EB' } },
    axisTick: { show: false },
    axisLabel: { color: '#86909C', fontSize: 11 },
  },
  yAxis: {
    type: 'value',
    splitLine: { lineStyle: { color: '#F2F3F5' } },
    axisLabel: { color: '#86909C', fontSize: 11 },
  },
  series: [
    {
      name: props.name,
      type: 'line',
      smooth: true,
      symbol: 'none',
      data: props.data.map((item) => item.hits),
      lineStyle: { width: 2, color: '#409EFF' },
      areaStyle: { color: 'rgba(64, 158, 255, 0.12)' },
    },
  ],
}))
</script>

<template>
  <VChart class="chart" :option="option" :style="{ height }" autoresize />
</template>

<style scoped>
.chart {
  width: 100%;
}
</style>
