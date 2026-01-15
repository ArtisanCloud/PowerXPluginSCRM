export default defineNuxtPlugin((nuxtApp) => {
  const csrf = useCookie<string | null>('px_csrf_token')
  if (!csrf.value) {
    return
  }
  nuxtApp.$fetch = $fetch.create({
    headers: {
      'X-CSRF-Token': csrf.value,
    },
  })
})
