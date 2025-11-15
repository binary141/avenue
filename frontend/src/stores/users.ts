import type { LoadableData } from "@/types/base";
import type { User } from "@/types/users";
import api, { setGlobalRequestHeader } from "@/utils/api";
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
    const token = ref<string | null>(
        JSON.parse(localStorage.getItem("token") || "null"),
    );

    function setToken(value: string | null) {
        token.value = value;
        localStorage.setItem("token", JSON.stringify(value));

        if (value !== null) {
            setGlobalRequestHeader("Authorization", `Token ${value}`);
        } else {
            setGlobalRequestHeader("Authorization", undefined);
        }
    }

    async function logIn(data: User) {
        userData.value.data = data;
        loggedIn.value = true;
    }

    async function logInAPI(userData: { username: string; password: string }) {
        const response = await api({
            url: "login",
            method: "POST",
            json: userData,
        });

        return response;
    }

    async function logOut() {
        // TODO: Implement logout logic
        loggedIn.value = false;
        userData.value.data = structuredClone(userDataDefault);
        setToken(null);
    }

    async function signUp(userData: {email: string; password:string }) {
        const response = await signUpAPI(userData);

        if (response.ok || response.status === 201) {
            setToken(response.body.session_id);
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
    async function updateUser(json: Partial<User>) {
        userData.value.loading = true;
        const response = await api({ url: `v1/user/${userData.value.data.id}`, method: 'PATCH', json})
        userData.value.loading = false;

        if (response.ok) {
            userData.value.data = response.body as User;
        } else {
            userData.value.error = response.body;
        }
        
        return response;
    }
    async function updatePassword(password: string) {
        // TODO: Implement password update logic
    }

    return {
        userData,
        loggedIn,
        token,
        logIn,
        logInAPI,
        logOut,
        signUpAPI,
        signUp,
        pullMe,
        updateUser,
        updatePassword,
        setToken,
    }
})