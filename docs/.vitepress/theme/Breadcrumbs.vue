<template>
  <nav class="breadcrumbs" aria-label="Хлебные крошки">
    <ol>
      <li v-for="(crumb, i) in crumbs" :key="crumb.link">
        <a v-if="crumb.link && i < crumbs.length - 1" :href="crumb.link">{{ crumb.text }}</a>
        <span v-else>{{ crumb.text }}</span>
        <span v-if="i < crumbs.length - 1" class="sep"> / </span>
      </li>
    </ol>
  </nav>
</template>

<script setup lang="ts">
import { useData, useRoute } from 'vitepress'
import { computed } from 'vue'

const { site, theme } = useData()
const route = useRoute()

const crumbs = computed(() => {
  const path = route.path
  const parts = path.split('/').filter(Boolean)
  const res: { text: string; link: string }[] = [{ text: 'Главная', link: '/' }]
  let acc = ''
  for (const p of parts) {
    acc += '/' + p
    // ищем в sidebar
    const label = findLabel(acc) || p
    res.push({ text: label, link: acc + '/' })
  }
  // последний — без линка
  if (res.length > 1) res[res.length - 1].link = ''
  return res
})

function findLabel(path: string): string | null {
  const sidebar: any = theme.value.sidebar
  for (const key in sidebar) {
    for (const group of sidebar[key]) {
      if (Array.isArray(group)) {
        for (const item of group) {
          if (item.link === path || item.link === path + '/') return item.text
        }
      } else if (group.link === path) {
        return group.text
      }
      if (group.items) {
        for (const item of group.items) {
          if (item.link === path) return item.text
        }
      }
    }
  }
  // fallback: capitalize
  const last = path.split('/').pop() || ''
  return last ? last.charAt(0).toUpperCase() + last.slice(1) : null
}
</script>

<style scoped>
.breadcrumbs { font-size: 13px; color: var(--vp-c-text-2); margin-bottom: 16px; }
.breadcrumbs ol { display: flex; flex-wrap: wrap; gap: 4px; list-style: none; padding: 0; margin: 0; }
.breadcrumbs a { color: var(--vp-c-brand-1); text-decoration: none; }
.breadcrumbs a:hover { text-decoration: underline; }
.sep { opacity: 0.5; }
</style>
