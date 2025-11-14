import { defineStore } from 'pinia'
import { ref } from 'vue';
import { useUsersStore } from './users';
import api from '@/utils/api';

export const useSiteStore = defineStore("site", () => {
  const userStore = useUsersStore();
  const appLaunchDataStatus = ref<"loading" | "loaded" | "error">("loading");

  async function loadAppLaunchData() {
    appLaunchDataStatus.value = "loading";

    const responseData = await api({ url: "app-launch/" });

    if (responseData.status === 200) {
      if (responseData.body.user_data !== undefined) {
        userStore.logIn();
      }

      document.documentElement.classList.remove("app-not-launched");
      appLaunchDataStatus.value = "loaded";
    } else if (responseData.status === 401) {
      // The user most likely had a DRF token stored that used to be valid but is
      // no longer valid. They probably changed their password on a different
      // device. Since a 401 error occurred, the response handler in main.ts
      // should have logged the user out, so we will try again now.

      loadAppLaunchData();
    } else {
      appLaunchDataStatus.value = "error";
    }
  }

  return {
    appLaunchDataStatus,
    loadAppLaunchData
  }
});