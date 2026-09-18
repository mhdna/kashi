<template>
  <div class="login-form">
    <div class="login-form-content">
      <AppLogo class="login-logo mx-auto" />

      <div class="text-subtitle-1 text-medium-emphasis">Account</div>

      <v-text-field
        v-model="username"
        density="compact"
        placeholder="Username"
        prepend-inner-icon="mdi-account-outline"
        variant="outlined"
        @keyup.enter="submit"
      />

      <div class="text-subtitle-1 text-medium-emphasis d-flex align-center justify-space-between">
        Password

        <!-- <a -->
        <!--   class="text-caption text-decoration-none text-blue" -->
        <!--   href="#" -->
        <!--   rel="noopener noreferrer" -->
        <!--   target="_blank" -->
        <!-- > -->
        <!--   Forgot login password?</a> -->
      </div>

      <v-text-field
        v-model="password"
        :append-inner-icon="visible ? 'mdi-eye-off' : 'mdi-eye'"
        density="compact"
        placeholder="Enter your password"
        prepend-inner-icon="mdi-lock-outline"
        :type="visible ? 'text' : 'password'"
        variant="outlined"
        @click:append-inner="visible = !visible"
        @keyup.enter="submit"
      />

      <v-btn
        block
        color="blue"
        :loading="submitting"
        size="large"
        variant="tonal"
        @click="submit"
      >
        Log In
      </v-btn>

      <!-- Feedback lives here, absolutely positioned under the card so an
           error message or the lockout warning can appear/disappear without
           ever changing the centered form's height or nudging it upward. -->
      <div class="login-feedback">
        <div aria-live="polite" role="alert">
          <v-alert v-if="error" density="compact" type="error" variant="tonal">
            {{ error }}
          </v-alert>
        </div>

        <v-card
          v-if="failedAttempts >= 3"
          class="mt-3"
          color="surface-variant"
          variant="tonal"
        >
          <v-card-text class="text-medium-emphasis text-caption">
            Warning: After 3 consecutive failed login attempts, your account will be temporarily locked for three hours.
            If you must login now, you can also click "Forgot login password?" below to reset the login password.
          </v-card-text>
        </v-card>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref } from 'vue'
  import { useRouter } from 'vue-router'
  import AppLogo from '@/components/AppLogo.vue'
  import { useAuthStore } from '@/stores/auth'

  const router = useRouter()
  const authStore = useAuthStore()

  const visible = ref(false)
  const username = ref('')
  const password = ref('')
  const submitting = ref(false)
  const error = ref('')
  const failedAttempts = ref(0)

  async function submit () {
    if (submitting.value) return

    error.value = ''
    if (!username.value.trim() || !password.value) {
      error.value = 'Enter your username and password.'
      return
    }

    submitting.value = true
    try {
      await authStore.login(username.value.trim(), password.value)
      router.push('/')
    } catch (error_) {
      // Only a rejected credential (400/401) counts toward the lockout
      // warning, not a server/network problem the user can't act on.
      if (error_.status === 400 || error_.status === 401) {
        failedAttempts.value += 1
      }
      error.value = error_.message
    } finally {
      submitting.value = false
    }
  }
</script>

<style scoped>
.login-form {
  align-items: center;
  box-sizing: border-box;
  display: flex;
  justify-content: center;
  min-height: 100dvh;
  padding: 24px 24px 72px;
  width: 100%;
}

.login-form-content {
  flex-shrink: 0;
  margin: auto;
  padding: 48px;
  position: relative;
  width: min(100%, 448px);
}

.login-logo {
  margin-bottom: 24px;
}

/* Pinned just below the card and taken out of flow, so the error message
   and lockout warning never affect the centered form's height or position. */
.login-feedback {
  position: absolute;
  top: 100%;
  left: 48px;
  right: 48px;
  margin-top: -16px;
}

@media (max-width: 600px) {
  .login-feedback {
    left: 0;
    right: 0;
  }
}

@media (max-width: 600px) {
  .login-form {
    padding-bottom: 56px;
  }

  .login-form-content {
    padding: 24px 0;
  }
}
</style>
