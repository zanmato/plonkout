import { useToast as usePrimeToast } from "primevue/usetoast";

export function useToast() {
  const toast = usePrimeToast();

  const showSuccess = (message, summary = "Success") => {
    toast.add({
      severity: "success",
      summary,
      detail: message,
      life: 3000,
    });
  };

  const showError = (message, summary = "Error") => {
    toast.add({
      severity: "error",
      summary,
      detail: message,
      life: 5000,
    });
  };

  const showWarning = (message, summary = "Warning") => {
    toast.add({
      severity: "warn",
      summary,
      detail: message,
      life: 4000,
    });
  };

  const showInfo = (message, summary = "Info") => {
    toast.add({
      severity: "info",
      summary,
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
