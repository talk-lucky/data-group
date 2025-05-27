const { defineConfig } = require('@vue/cli-service')
module.exports = defineConfig({
  transpileDependencies: true,
  lintOnSave: false, // Can be true if preferred
  // Vuetify plugin options if using vue-cli-plugin-vuetify
  // pluginOptions: {
  //   vuetify: {
  //     // https://github.com/vuetifyjs/vue-cli-plugin-vuetify/tree/next/packages/vue-cli-plugin-vuetify#configuration
  //   }
  // }
})
