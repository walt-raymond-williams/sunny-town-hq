import { createRouter, createWebHistory } from 'vue-router'
import StudentPage from './features/student/StudentPage.vue'
import TeacherPage from './features/teacher/TeacherPage.vue'
import SplashPage from './pages/SplashPage.vue'
import SunnyTownPage from './pages/SunnyTownPage.vue'
import TeacherLoginPage from './pages/TeacherLoginPage.vue'

export const routes = [
  {
    path: '/',
    name: 'splash',
    component: SplashPage,
  },
  {
    path: '/student',
    name: 'student',
    component: StudentPage,
  },
  {
    path: '/student/pet/sunny-town',
    name: 'sunny-town',
    component: SunnyTownPage,
  },
  {
    path: '/teacher/login',
    name: 'teacher-login',
    component: TeacherLoginPage,
  },
  {
    path: '/teacher',
    name: 'teacher',
    component: TeacherPage,
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
