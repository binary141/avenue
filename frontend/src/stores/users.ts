import type { LoadableData } from "@/types/base";
import type { User } from "@/types/users";
import api from "@/utils/api";
import { defineStore } from "pinia";
import { ref } from "vue";

const userDataDefault: User = {
    id: 1
}

export const useUsersStore = defineStore('users', () => {
    const userData = ref<LoadableData<User>>({
        data: userDataDefault,
        loading: false,
    })
    const loggedIn = ref(false);


    async function logIn() {
        // TODO: Implement login logic
    }

    async function logOut() {
        // TODO: Implement logout logic
        loggedIn.value = false;
        userData.value.data = structuredClone(userDataDefault);
    }

    async function signUp() {
        // TODO: Implement sign-up logic
    }

    async function pullUser(id: number) {
        userData.value.loading = true;
        // TODO: Update url based on API endpoint
        const response = await api({ url: `users/${id}/` })
        userData.value.loading = false;

        if (response.ok) {
            userData.value.data = response.body as User;
        } else {
            userData.value.error = response.body;
        }
    }
    async function updateUser(json: Partial<User>) {
        userData.value.loading = true;
        const response = await api({ url: '', method: 'PATCH', json})
        userData.value.loading = false;

        if (response.ok) {
            userData.value.data = response.body as User;
        } else {
            userData.value.error = response.body;
        }
        
        return response;
    }

    return {
        userData,
        logIn,
        logOut,
        signUp,
        pullUser,
        updateUser,
    }
})