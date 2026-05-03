import { createRouter, createWebHistory } from 'vue-router'
import UsersView from '../views/UsersView.vue'
import MessagesView from '../views/MessagesView.vue'
import ContactsView from '../views/ContactsView.vue'
import RequestsView from '../views/RequestsView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/users' },
    { path: '/users', component: UsersView },
    { path: '/messages', component: MessagesView },
    { path: '/contacts', component: ContactsView },
    { path: '/requests', component: RequestsView },
  ],
})

export default router
