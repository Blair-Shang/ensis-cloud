<script setup lang="ts">
import { computed } from 'vue'
import { renderUsageChartSvg, type UsageChartSpec } from './usage-chart'

const props = defineProps<{
  spec: UsageChartSpec
  empty?: string
}>()

const hasData = computed(() => props.spec.values.some((v) => v > 0))
const svg = computed(() => (hasData.value ? renderUsageChartSvg(props.spec) : ''))
</script>

<template>
  <div class="ai-chart">
    <div v-if="hasData" class="ai-chart__svg" v-html="svg" />
    <div v-else class="ai-chart__empty">{{ empty || '暂无数据' }}</div>
  </div>
</template>

<style scoped>
.ai-chart {
  min-height: 220px;
  color: rgba(0, 0, 0, 0.65);
}

.ai-chart__svg :deep(svg) {
  display: block;
  width: 100%;
  height: auto;
  max-height: 240px;
}

.ai-chart__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 220px;
  color: var(--ops-muted);
  font-size: 13px;
}
</style>
