/**
 * Ops 用的 @niuma/ui 薄封装：只 re-export 需要的 Rs*。
 * 从包入口具名导入。宿主启用 niumaUiHost：dev 改到源码文件，build 改到 dist 子路径。
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
} from '@niuma/ui'
export type { RsTableColumn } from '@niuma/ui'
