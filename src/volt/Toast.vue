<template>
  <Toast
    unstyled
    :pt="theme"
    :ptOptions="{
      mergeProps: ptViewMerge,
    }"
  >
    <template #closeicon>
      <TimesIcon />
    </template>
    <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
      <slot :name="slotName" v-bind="slotProps ?? {}" />
    </template>
  </Toast>
</template>

<script setup>
import TimesIcon from "@primevue/icons/times";
import Toast from "primevue/toast";
import { ref } from "vue";
import { ptViewMerge } from "./utils";

const theme = ref({
  root: `w-96 whitespace-pre-line break-words
        p-top-center:-translate-x-1/2 p-bottom-center:-translate-x-1/2
        p-center:min-w-[20vw] p-center:-translate-x-1/2 p-center:-translate-y-1/2`,
  message: `mb-4 border-2 border-black rounded-lg shadow-[4px_4px_0px_0px_#000] font-semibold
        p-info:bg-blue-100 p-info:text-black
        p-success:bg-green-100 p-success:text-black
        p-warn:bg-yellow-100 p-warn:text-black
        p-error:bg-red-100 p-error:text-black
        p-secondary:bg-gray-100 p-secondary:text-black
        p-contrast:bg-white p-contrast:text-black`,
  messageContent: `flex items-start p-4 gap-3`,
  messageIcon: `flex-shrink-0 text-xl w-6 h-6 mt-0.5 font-bold`,
  messageText: `flex-auto flex flex-col gap-2`,
  summary: `font-bold text-lg`,
  detail: `font-semibold text-base text-black`,
  buttonContainer: ``,
  closeButton: `flex items-center justify-center overflow-hidden relative cursor-pointer bg-black text-white select-none
        transition-all duration-100 w-8 h-8 rounded-md border-2 border-black shadow-[2px_2px_0px_0px_#000] -mt-2 -me-2 p-0
        hover:translate-x-[-1px] hover:translate-y-[-1px] hover:shadow-[3px_3px_0px_0px_#000]
        active:translate-x-[1px] active:translate-y-[1px] active:shadow-[1px_1px_0px_0px_#000]`,
  closeIcon: `text-base w-4 h-4 font-bold`,
  transition: {
    enterFromClass: "opacity-0 translate-x-4 scale-95",
    enterActiveClass: "transition-all duration-200 ease-out",
    leaveFromClass: "max-h-[1000px]",
    leaveActiveClass: "transition-all duration-200 ease-in",
    leaveToClass:
      "max-h-0 opacity-0 mb-0 overflow-hidden translate-x-4 scale-95",
  },
});
</script>
