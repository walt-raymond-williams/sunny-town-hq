import { createRouter, createWebHistory } from 'vue-router'

export const routes = [
  {
    path: '/',
    name: 'splash',
    component: {},
  },
  {
    path: '/student',
    name: 'student',
    component: {},
  },
  {
    path: '/teacher/login',
    name: 'teacher-login',
    component: {},
  },
  {
    path: '/teacher',
    name: 'teacher',
    component: {},
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
