<template>
  <img
    class="brand-logo"
    :src="logoSrc"
    :alt="altText"
    :style="{ width: normalizedWidth }"
  >
</template>

<script>
import brand from '../constants/brand'
import logoSrc from '../assets/meilu-logo.svg'

export default {
  name: 'BrandLogo',
  props: {
    altText: {
      type: String,
      default: `${brand.productName} Logo`
    },
    width: {
      type: [String, Number],
      default: 260
    }
  },
  computed: {
    normalizedWidth() {
      return typeof this.width === 'number' ? `${this.width}px` : this.width
    }
  },
  data() {
    return {
      logoSrc
    }
  }
}
</script>

<style scoped>
.brand-logo {
  display: block;
  max-width: 100%;
  height: auto;
}
</style>
