<script setup lang="ts">
import { computed } from 'vue'
import { RsButton } from '@/components/ui'

const props = withDefaults(
  defineProps<{
    total: number
    page: number
    pageSize?: number
  }>(),
  { pageSize: 20 },
)

const emit = defineEmits<{
  change: [page: number]
}>()

const pages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const from = computed(() => (props.total === 0 ? 0 : (props.page - 1) * props.pageSize + 1))
const to = computed(() => Math.min(props.total, props.page * props.pageSize))

function go(next: number) {
  if (next < 1 || next > pages.value || next === props.page) return
  emit('change', next)
}
</script>

<template>
  <div class="ops-pager">
    <span class="ops-muted">第 {{ from }}–{{ to }} 条，共 {{ total }} 条</span>
    <div class="ops-pager__btns">
      <RsButton variant="secondary" size="sm" :disabled="page <= 1" @click="go(page - 1)">上一页</RsButton>
      <span class="ops-muted">{{ page }} / {{ pages }}</span>
      <RsButton variant="secondary" size="sm" :disabled="page >= pages" @click="go(page + 1)">下一页</RsButton>
    </div>
  </div>
</template>
