import { createRouter, createWebHistory } from "vue-router";
import { useUsersStore } from "@/stores/users";

const scrollPositions = Object(null);

const router = createRouter({
    history: createWebHistory(),
    routes: [
        { path: '/', name: 'home', component: () => import('@/views/DrivePage.vue') },
        {
            path: '/login',
            name: 'login',
            component: () => import('@/views/LoginPage.vue'),
            meta: { allowAnonymous: true },
        },
        {
            path: '/signup',
            name:'signup',
            component: () => import('@/views/SignUpPage.vue'),
            meta: { allowAnonymous: true },
        },
        {
            path: '/drive',
            name: 'drive',
            component: () => import('@/views/DrivePage.vue'),
        },
        {
            path: '/logout',
            name: 'logout',
            component: () => import('@/views/LogoutPage.vue')
        },
    ],
    scrollBehavior(to, from, savedPosition) {
        if (savedPosition && scrollPositions[to.path] !== undefined) {
            document.documentElement.scrollTop = scrollPositions[to.path];
        } else {
            document.documentElement.scrollTop = 0;
        }
    },
});

router.beforeEach(async (to, from, next) => {
    const store = useUsersStore();

    if (document.documentElement.classList.contains("app-not-launched")) {
        await new Promise<void>((resolve) => {
            const observer = new MutationObserver(() => {
                observer.disconnect();
                resolve();
            });

            observer.observe(document.documentElement, { attributes: true });
        });
    }

    if (
        to.matched.some((record) => record.meta.allowAnonymous) ||
        store.loggedIn
    ) {
        if (from.path) {
            scrollPositions[from.path] = document.documentElement.scrollTop;
        }

        next();
    }  else {
        next({
            name: "login",
            query: { next: to.fullPath },
        });
    }
});

export default router;
