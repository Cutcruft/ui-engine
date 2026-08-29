export default {
  title: 'ui-engine',
  description: 'Go + WASM + YAML — декларативные интерфейсы',
  lang: 'ru-RU',
  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: 'Гайд', link: '/guide/' },
      { text: 'API', link: '/api/cli' },
      { text: 'Примеры', link: '/guide/examples' },
      { text: 'GitHub', link: 'https://github.com/ui-engine/ui-engine' }
    ],
    sidebar: {
      '/guide/': [
        { text: 'Введение', link: '/guide/' },
        { text: 'Установка', link: '/guide/installation' },
        { text: 'Быстрый старт', link: '/guide/quickstart' },
        { text: 'Конфиги', link: '/guide/configs' },
        { text: 'Компоненты', link: '/guide/components' },
        { text: 'Модули', link: '/guide/modules' },
        { text: 'Анимации', link: '/guide/animations' },
        { text: 'Темизация', link: '/guide/theming' },
        { text: 'JS Bridge', link: '/guide/js-bridge' },
        { text: 'Примеры', link: '/guide/examples' }
      ],
      '/api/': [
        { text: 'CLI', link: '/api/cli' },
        { text: 'Конфиги', link: '/api/configs' },
        { text: 'Runtime', link: '/api/runtime' }
      ]
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/ui-engine/ui-engine' }
    ],
    footer: {
      message: 'MIT Licensed',
      copyright: 'Copyright © 2024 ui-engine'
    }
  },
  vite: {
    css: {
      preprocessorOptions: {
        css: {
          additionalData: `:root { --vp-c-brand-1: #6366f1; --vp-c-brand-2: #4f46e5; }`
        }
      }
    }
  }
}
