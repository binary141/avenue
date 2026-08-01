import type { LoadableData } from "@/types/base";
import type { User } from "@/types/users";
import api from "@/utils/api";
import { defineStore } from "pinia";
import { ref } from "vue";

const userDataDefault: User = {
    id: 1,
    canLogin: false,
    isAdmin: false,
    quota: 0,
    spaceUsed: 0,
    createdAt: '',
    deletedAt: null,
    email: '',
    firstName: null,
    lastName: null,
    updatedAt: null,
}

export const useUsersStore = defineStore('users', () => {
    const userData = ref<LoadableData<User>>({
        data: userDataDefault,
        loading: false,
    })
    const loggedIn = ref(false);
    const fileSharingEnabled = ref(false);
    const folderSharingEnabled = ref(false);

    async function logIn(data: User) {
        userData.value.data = data;
        loggedIn.value = true;
    }

    // Clears local session state without calling the backend. Use this when
    // a request already told us the session is gone (e.g. a 401), so there's
    // nothing left to invalidate server-side.
    function resetSession() {
        loggedIn.value = false;
        userData.value.data = structuredClone(userDataDefault);
    }

    async function logInAPI(userData: { email: string; password: string }) {
        const response = await api({
            url: "login",
            method: "POST",
            json: userData,
        });

        return response;
    }

    async function createUser(userData: { email: string; firstName: string; lastName: string; password?: string | null; quota: number; isAdmin?: boolean; sendEmail?: boolean }) {
        const response = await api({
            url: "v1/user",
            method: "POST",
            json: userData,
        });

        return response;
    }

    async function getOAuthProviders() {
        const response = await api({
            url: "auth/providers",
            method: "GET",
        });

        return response;
    }

    async function exchangeOAuthCode(code: string) {
        const response = await api({
            url: "auth/exchange",
            method: "POST",
            json: { code },
        });

        return response;
    }

    async function logOut() {
        const response = await api({
            url: "v1/logout",
            method: "POST",
        });

        if (!response.ok && response.status != 401) {
          return;
        }

        resetSession();

        return response;
    }

    async function signUp(userData: {email: string; password:string }) {
        const response = await signUpAPI(userData);

        if (response.ok || response.status === 201) {
            await logIn(response.body.user_data);
        }

    return response;
    }

    async function signUpAPI(userData: { email: string; password: string }) {
        const response = await api({
            url: "register",
            method: "POST",
            json: userData,
        });

    return response;
    }

    async function getUsers() {
        const response = await api({
            url: "v1/users",
            method: "GET",
        });

    return response;
    }

    async function pullMe() {
        userData.value.loading = true;
        // TODO: Update url based on API endpoint
        const response = await api({ url: 'v1/user/profile' })
        userData.value.loading = false;

        if (response.ok) {
            userData.value.data = response.body as User;
        } else {
            userData.value.error = response.body;
        }

        return response;
    }
    async function updateUser(req: Partial<User>) {
        userData.value.loading = true;
        const response = await api({ url: `v1/user/${userData.value.data.id}`, method: 'PATCH', json: req})
        userData.value.loading = false;

        if (response.ok) {
            userData.value.data = response.body as User;
        } else {
            userData.value.error = response.body;
        }

        return response;
    }
    async function updatePassword() {
        // TODO: Implement password update logic
    }

    return {
        userData,
        getUsers,
        createUser,
        loggedIn,
        fileSharingEnabled,
        folderSharingEnabled,
        logIn,
        resetSession,
        logInAPI,
        getOAuthProviders,
        exchangeOAuthCode,
        logOut,
        signUpAPI,
        signUp,
        pullMe,
        updateUser,
        updatePassword,
    }
})
