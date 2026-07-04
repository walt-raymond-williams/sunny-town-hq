import { createRouter, createWebHistory } from 'vue-router'

export const routes = [
  {
    path: '/',
    name: 'splash',
    component: () => import('./pages/SplashPage.vue'),
  },
  {
    path: '/student',
    name: 'student',
    component: () => import('./features/student/StudentPage.vue'),
  },
  {
    path: '/student/pet/sunny-town',
    name: 'sunny-town',
    component: () => import('./features/sunny-town/SunnyTownPage.vue'),
  },
  {
    path: '/student/pet/sunny-town/canvas',
    name: 'sunny-town-canvas',
    component: () => import('./features/sunny-town/SunnyTownPage.vue'),
  },
  {
    path: '/student/pet/sunny-town/godot',
    name: 'sunny-town-godot',
    component: () => import('./features/sunny-town/SunnyTownGodotPage.vue'),
  },
  {
    path: '/teacher/login',
    name: 'teacher-login',
    component: () => import('./pages/TeacherLoginPage.vue'),
  },
  {
    path: '/teacher',
    name: 'teacher',
    component: () => import('./features/teacher/TeacherPage.vue'),
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/',
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
