<script setup lang="ts">
import { RsConfigProvider } from '@/components/ui'
</script>

<template>
  <RsConfigProvider theme="light" locale="zh-CN">
    <router-view />
  </RsConfigProvider>
</template>
