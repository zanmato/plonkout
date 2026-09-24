/**
 * Minimal typings for vue3-virtual-scroller, which ships none.
 * Only covers the props and slots the app uses.
 */
declare module "vue3-virtual-scroller" {
  import type {
    AllowedComponentProps,
    ComponentCustomProps,
    VNodeChild,
    VNodeProps,
  } from "vue";

  type BaseProps = VNodeProps & AllowedComponentProps & ComponentCustomProps;

  export interface DynamicScrollerSlotScope<T> {
    item: T;
    index: number;
    active: boolean;
  }

  export interface DynamicScrollerProps<T> {
    items: T[];
    minItemSize: number | string;
    keyField?: string;
    direction?: "vertical" | "horizontal";
    buffer?: number;
    pageMode?: boolean;
  }

  export const DynamicScroller: new <T>(
    props: DynamicScrollerProps<T> & BaseProps,
  ) => {
    $props: DynamicScrollerProps<T> & BaseProps;
    $slots: {
      default?: (scope: DynamicScrollerSlotScope<T>) => VNodeChild;
      before?: () => VNodeChild;
      after?: () => VNodeChild;
    };
  };

  export interface DynamicScrollerItemProps {
    item: unknown;
    active: boolean;
    sizeDependencies?: unknown[];
    watchData?: boolean;
    tag?: string;
    emitResize?: boolean;
  }

  export const DynamicScrollerItem: new (
    props: DynamicScrollerItemProps & BaseProps,
  ) => {
    $props: DynamicScrollerItemProps & BaseProps;
    $slots: {
      default?: () => VNodeChild;
    };
  };
}
