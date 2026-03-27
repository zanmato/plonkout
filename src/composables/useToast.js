import { useToast as usePrimeToast } from "primevue/usetoast";

export function useToast() {
  const toast = usePrimeToast();

  const showSuccess = (message) => {
    toast.add({
      severity: "success",
      detail: message,
      life: 3000,
    });
  };

  const showError = (message) => {
    toast.add({
      severity: "error",
      detail: message,
      life: 5000,
    });
  };

  const showWarning = (message) => {
    toast.add({
      severity: "warn",
      detail: message,
      life: 4000,
    });
  };

  const showInfo = (message) => {
    toast.add({
      severity: "info",
      detail: message,
      life: 3000,
    });
  };

  return {
    showSuccess,
    showError,
    showWarning,
    showInfo,
  };
}
