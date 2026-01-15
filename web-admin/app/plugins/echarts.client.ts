import { defineNuxtPlugin } from '#app'
import VChart from 'vue-echarts'

export default defineNuxtPlugin((nuxtApp) => {
  // 注册 VChart 组件
  nuxtApp.vueApp.component('VChart', VChart)
})