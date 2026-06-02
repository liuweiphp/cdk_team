<template>
  <div class="user-layout">
    <header class="user-header glass-panel">
      <div class="header-inner">
        <h1 class="logo"><span class="accent">CDK</span> Exchange</h1>
        <nav class="user-nav">
          <router-link to="/exchange">兑换文本</router-link>
          <router-link to="/login">后台登录</router-link>
        </nav>
      </div>
    </header>
    <main class="user-main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts"></script>

<style scoped lang="scss">
.user-layout {
  min-height: 100vh;
  background: radial-gradient(ellipse at 50% 0%, rgba(34,197,94,0.06) 0%, transparent 70%),
              var(--bg-deep);
}

.user-header {
  position: sticky;
  top: 12px;
  margin: 12px 16px;
  z-index: 100;
  .header-inner {
    max-width: 1200px;
    margin: 0 auto;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 24px;
  }
}

.logo {
  font-family: var(--font-heading);
  font-size: 20px;
  font-weight: 700;
  color: var(--foreground);
  .accent { color: var(--accent); }
}

.user-nav {
  display: flex;
  align-items: center;
  gap: 20px;
  a {
    color: var(--foreground-muted);
    text-decoration: none;
    font-size: 14px;
    transition: color 0.2s;
    &:hover, &.router-link-active { color: var(--accent); }
  }
}

.user-main {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px 16px;
}
</style>
