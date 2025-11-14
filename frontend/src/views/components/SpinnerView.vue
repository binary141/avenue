<template>
  <svg
    v-show="show"
    :width="sizeToUse"
    :height="sizeToUse"
    :style="spinnerStyle"
    viewBox="0 0 38 38"
    xmlns="http://www.w3.org/2000/svg"
    class="spinner"
  >
    <g fill="none" fill-rule="evenodd">
      <g transform="translate(1 1)">
        <path
          d="M36 18c0-9.94-8.06-18-18-18"
          :stroke="colorToUse"
          :stroke-width="thicknessToUse"
        >
          <animateTransform
            attributeName="transform"
            type="rotate"
            from="0 18 18"
            to="360 18 18"
            :dur="`${speedToUse}s`"
            repeatCount="indefinite"
          />
        </path>
      </g>
    </g>
  </svg>
</template>

<script lang="ts">
import { computed, defineComponent, type PropType, toRaw } from "vue";

export default defineComponent({
  props: {
    show: {
      type: Boolean,
      required: false,
      default: true,
    },
    size: {
      type: Number,
      required: false,
      default: 18,
    },
    thickness: {
      type: Number,
      required: false,
      default: 1.5,
    },
    speed: {
      type: Number,
      required: false,
      default: 0.7,
    },
    center: {
      type: Boolean,
      required: false,
      default: false,
    },
    color: {
      type: String,
      required: false,
      default: "#000",
    },
    preset: {
      type: String as PropType<"large">,
      required: false,
      default: undefined,
    },
  },
  setup(props) {
    let center = false;
    let color: string;
    let size: number;
    let thickness: number;
    let speed: number;

    if (props.preset === "large") {
      center = true;
      color = "#fff";
      size = 64;
      ({ thickness, speed } = toRaw(props));
    } else {
      ({ center, color, size, thickness, speed } = toRaw(props));
    }

    const spinnerStyle = computed(() => ({
      display: center ? "block" : "inline-block",
      margin: center ? "0 auto" : "0",
    }));

    return {
      colorToUse: color,
      sizeToUse: size,
      thicknessToUse: thickness,
      speedToUse: speed,
      spinnerStyle,
    };
  },
});
</script>

<style scoped>
.spinner {
  vertical-align: text-bottom;
}
</style>
