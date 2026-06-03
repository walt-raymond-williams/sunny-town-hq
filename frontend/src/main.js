import { createApp } from 'vue'
import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import { aliases, mdi } from 'vuetify/iconsets/mdi'
import './style.css'
import App from './App.vue'

const vuetify = createVuetify({
  components,
  directives,
  icons: {
    defaultSet: 'mdi',
    aliases,
    sets: {
      mdi,
    },
  },
  theme: {
    defaultTheme: 'hqLight',
    themes: {
      hqLight: {
        dark: false,
        colors: {
          background: '#f3f6f7',
          surface: '#ffffff',
          primary: '#27746f',
          secondary: '#17212b',
          success: '#24823f',
          error: '#9f2d2d',
        },
      },
    },
  },
})

createApp(App).use(vuetify).mount('#app')
