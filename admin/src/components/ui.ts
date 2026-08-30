/**
 * Ops 用的 @niuma/ui 薄封装：只 re-export 需要的 Rs*。
 * 从包入口具名导入，由打包器摇掉 Monaco / xterm。
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
