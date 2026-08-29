/**
 * 从 CHANGELOG.md 抽出指定版本段落，写成 GitHub Release 正文。
 * 用法: node .github/scripts/changelog-notes.mjs <version> [outfile] [--require]
 */
import fs from 'node:fs'
import path from 'node:path'

const args = process.argv.slice(2).filter((a) => a !== '--require')
const requireSection = process.argv.includes('--require')
const version = args[0]
const outfile = args[1] || 'release-notes.md'

if (!version) {
  process.stderr.write('usage: changelog-notes.mjs <version> [outfile] [--require]\n')
  process.exit(1)
}

const changelogPath = path.resolve('CHANGELOG.md')
const md = fs.readFileSync(changelogPath, 'utf8')
const escaped = version.replace(/\./g, '\\.')
const heading = md.match(new RegExp(`^## \\[${escaped}\\][^\\n]*$`, 'm'))
const match = md.match(
  new RegExp(`## \\[${escaped}\\][^\\n]*\\n([\\s\\S]*?)(?=\\n## \\[|\\n\\[[^\\]]+\\]:|$)`),
)
const section = (match ? match[1] : '').trim()

if (requireSection && (!heading || !section)) {
  process.stderr.write(
    `CHANGELOG.md 缺少版本段落 ## [${version}]（或该段为空）。发版前请从 [Unreleased] 迁入并写明日期。\n`,
  )
  process.exit(1)
}

const dateMatch = heading ? heading[0].match(/## \[[^\]]+\] - (\d{4}-\d{2}-\d{2})/) : null
const title = dateMatch ? `niuma-cloud v${version} (${dateMatch[1]})` : `niuma-cloud v${version}`
const body = [
  `## ${title}`,
  '',
  section || `_本版本暂无 CHANGELOG 条目。_`,
  '',
  '---',
  '',
  '### 产物',
  '',
  `- \`niuma-cloud-${version}-linux-amd64.tar.gz\``,
  `- \`niuma-cloud-${version}-windows-amd64.zip\``,
  '- `SHA256SUMS.txt`',
  '',
  `完整记录见 [CHANGELOG.md](https://github.com/Blair-Shang/niuma-cloud/blob/v${version}/CHANGELOG.md)。`,
].join('\n')

fs.writeFileSync(outfile, `${body}\n`, 'utf8')
process.stdout.write(`${body}\n`)
