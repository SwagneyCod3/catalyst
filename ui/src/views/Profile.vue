<template>
  <v-main v-if="userdata">
    <user-data-editor :userdata="userdata" @save="saveUserData"></user-data-editor>
  </v-main>
</template>

<script lang="ts">
import Vue from "vue";
import { UserData } from "@/client";
import { API } from "@/services/api";
import UserDataEditor from "@/components/UserDataEditor.vue";

interface State {
  userdata?: UserData;
}

export default Vue.extend({
  name: "Profile",
  data: (): State => ({
    userdata: undefined,
  }),
  components: {
    UserDataEditor,
  },
  watch: {
    $route: function () {
      this.loadUserData();
    },
  },
  methods: {
    saveUserData: function(userdata: UserData) {
      API.updateCurrentUserData(userdata).then(() => {
        this.$store.dispatch("alertSuccess", { name: "User data saved" });
      });
    },
    loadUserData: function () {
      API.currentUserData().then((response) => {
        this.userdata = response.data;
      });
    }
  },
  mounted() {
    this.loadUserData();
  }
});
</script>

