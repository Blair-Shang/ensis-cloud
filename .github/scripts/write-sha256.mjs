/**
 * 按 GNU coreutils 文本格式写出 SHA256SUMS.txt（hash 后两个空格再跟文件名）。
 * 用法: node .github/scripts/write-sha256.mjs <output-dir>
 */
import { createHash } from 'node:crypto'
import { readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const root = process.argv[2]
if (!root) {
  process.stderr.write('usage: write-sha256.mjs <output-dir>\n')
  process.exit(1)
}

const keep = /\.(tar\.gz|tgz|zip)$/i
const names = readdirSync(root)
  .filter((name) => {
    const full = join(root, name)
    return statSync(full).isFile() && keep.test(name)
  })
  .sort((a, b) => a.localeCompare(b))

const lines = names.map((name) => {
  const hash = createHash('sha256').update(readFileSync(join(root, name))).digest('hex')
  return `${hash}  ${name}`
})

const out = join(root, 'SHA256SUMS.txt')
writeFileSync(out, `${lines.join('\n')}${lines.length ? '\n' : ''}`, 'utf8')
process.stdout.write(`wrote ${lines.length} checksums -> ${out}\n`)
