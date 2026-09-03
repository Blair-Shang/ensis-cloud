/**
 * Ops 用的 @niuma/ui 薄封装：只 re-export 需要的 Rs*。
 * 从包入口具名导入。niumaUiHost 只服务 pnpm dev；vite build 走包主入口。
 */
export {
  RsConfigProvider,
  RsButton,
  RsInput,
  RsBadge,
  RsSelect,
  RsTable,
  RsDialog,
  RsConfirmDialog,
  RsCheckbox,
  RsLabel,
  RsMarkdown,
  RsTooltip,
  RsTooltipProvider,
} from '@niuma/ui'
export type { RsSelectModelValue, RsTableColumn } from '@niuma/ui'
