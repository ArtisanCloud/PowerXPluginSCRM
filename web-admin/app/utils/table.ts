import { computed } from "vue";
import type { ColumnDef } from "@tanstack/vue-table";

type LegacyColumn = {
  key?: string;
  label?: string;
  header?: ColumnDef<any>["header"];
  accessorKey?: string;
  id?: string;
} & Record<string, any>;

export const normalizeColumns = (columns: LegacyColumn[]): ColumnDef<any, any>[] =>
  columns.map((column) => {
    const accessorKey = column.accessorKey ?? column.key;
    const header = column.header ?? column.label;
    const id = column.id ?? accessorKey ?? column.key;
    return {
      ...column,
      accessorKey,
      header,
      id,
    } as ColumnDef<any, any>;
  });

export const useNormalizedColumns = (columns: LegacyColumn[]) =>
  computed(() => normalizeColumns(columns));
