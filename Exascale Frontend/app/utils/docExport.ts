/**
 * docExport — turn LLM-generated Markdown into a downloadable document: Markdown, PDF, Word (.docx),
 * PowerPoint (.pptx), or Excel (.xlsx). Everything runs in the browser; the heavy libraries are loaded
 * on demand (dynamic import) so they never touch the SSR path or the main bundle. The model emits
 * Markdown as the canonical form; each exporter parses it into blocks and renders the target format.
 */
export type DocFormat = 'markdown' | 'pdf' | 'docx' | 'pptx' | 'xlsx'

/** A parsed Markdown block. Tables carry their rows; everything else carries inline-stripped text. */
interface Block {
  type: 'h1' | 'h2' | 'h3' | 'p' | 'li' | 'table'
  text?: string
  rows?: string[][]
}

/** stripInline removes the Markdown emphasis/code/link syntax we don't carry into the rendered doc. */
function stripInline(s: string): string {
  return s
    .replace(/\*\*(.+?)\*\*/g, '$1')
    .replace(/\*(.+?)\*/g, '$1')
    .replace(/`(.+?)`/g, '$1')
    .replace(/\[(.+?)\]\([^)]*\)/g, '$1')
    .trim()
}

/** parseMarkdown converts Markdown into an ordered list of blocks (line-based — robust for LLM output). */
function parseMarkdown(md: string): Block[] {
  const blocks: Block[] = []
  let para: string[] = []
  let table: string[][] = []
  const flushPara = () => { if (para.length) { blocks.push({ type: 'p', text: para.join(' ') }); para = [] } }
  const flushTable = () => { if (table.length) { blocks.push({ type: 'table', rows: table }); table = [] } }
  for (const raw of md.replace(/\r\n/g, '\n').split('\n')) {
    const line = raw.trimEnd()
    if (/^\s*\|.*\|\s*$/.test(line)) {
      const cells = line.trim().replace(/^\|/, '').replace(/\|$/, '').split('|').map((c) => stripInline(c))
      if (cells.every((c) => /^:?-{2,}:?$/.test(c.trim()))) continue // separator row
      flushPara()
      table.push(cells)
      continue
    }
    flushTable()
    if (!line.trim()) { flushPara(); continue }
    const h = /^(#{1,3})\s+(.*)$/.exec(line)
    if (h) { flushPara(); blocks.push({ type: `h${h[1]!.length}` as Block['type'], text: stripInline(h[2]!) }); continue }
    const li = /^\s*[-*+]\s+(.*)$/.exec(line) ?? /^\s*\d+\.\s+(.*)$/.exec(line)
    if (li) { flushPara(); blocks.push({ type: 'li', text: stripInline(li[1]!) }); continue }
    para.push(stripInline(line))
  }
  flushPara()
  flushTable()
  return blocks
}

/** saveBlob triggers a browser download of bytes under filename. */
function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 3000)
}

/** safeName makes a filesystem-friendly base name from a title. */
function safeName(s: string): string {
  const n = (s || 'document').replace(/[^\w -]+/g, '').trim().replace(/\s+/g, '-').slice(0, 48)
  return n || 'document'
}

// ── PDF (jsPDF) ───────────────────────────────────────────────────────────────────────────────────
async function exportPDF(blocks: Block[], name: string) {
  const { jsPDF } = await import('jspdf')
  const doc = new jsPDF({ unit: 'pt', format: 'a4' })
  const margin = 50
  const pageW = doc.internal.pageSize.getWidth()
  const pageH = doc.internal.pageSize.getHeight()
  let y = margin
  const ensure = (h: number) => { if (y + h > pageH - margin) { doc.addPage(); y = margin } }
  const write = (text: string, size: number, bold: boolean, indent = 0, gap = 4) => {
    doc.setFont('helvetica', bold ? 'bold' : 'normal')
    doc.setFontSize(size)
    for (const ln of doc.splitTextToSize(text, pageW - margin * 2 - indent)) {
      ensure(size + 4)
      doc.text(ln, margin + indent, y)
      y += size + 4
    }
    y += gap
  }
  for (const b of blocks) {
    if (b.type === 'h1') write(b.text ?? '', 20, true, 0, 8)
    else if (b.type === 'h2') write(b.text ?? '', 15, true, 0, 6)
    else if (b.type === 'h3') write(b.text ?? '', 12.5, true, 0, 4)
    else if (b.type === 'li') write('•  ' + (b.text ?? ''), 11, false, 14, 2)
    else if (b.type === 'table') { for (const r of b.rows ?? []) write(r.join('   |   '), 10, r === b.rows?.[0], 0, 1) ; y += 4 }
    else write(b.text ?? '', 11, false, 0, 6)
  }
  doc.save(name + '.pdf')
}

// ── Word (.docx) ──────────────────────────────────────────────────────────────────────────────────
async function exportDocx(blocks: Block[], name: string) {
  const { Document, Packer, Paragraph, HeadingLevel, TextRun, Table, TableRow, TableCell, WidthType } = await import('docx')
  const children: Array<InstanceType<typeof Paragraph> | InstanceType<typeof Table>> = []
  for (const b of blocks) {
    if (b.type === 'h1') children.push(new Paragraph({ text: b.text ?? '', heading: HeadingLevel.HEADING_1 }))
    else if (b.type === 'h2') children.push(new Paragraph({ text: b.text ?? '', heading: HeadingLevel.HEADING_2 }))
    else if (b.type === 'h3') children.push(new Paragraph({ text: b.text ?? '', heading: HeadingLevel.HEADING_3 }))
    else if (b.type === 'li') children.push(new Paragraph({ text: b.text ?? '', bullet: { level: 0 } }))
    else if (b.type === 'table' && b.rows?.length) {
      children.push(new Table({
        width: { size: 100, type: WidthType.PERCENTAGE },
        rows: b.rows.map((r) => new TableRow({ children: r.map((c) => new TableCell({ children: [new Paragraph(c)] })) })),
      }))
    } else children.push(new Paragraph({ children: [new TextRun(b.text ?? '')] }))
  }
  const doc = new Document({ sections: [{ children }] })
  saveBlob(await Packer.toBlob(doc), name + '.docx')
}

// ── PowerPoint (.pptx) ────────────────────────────────────────────────────────────────────────────
async function exportPptx(blocks: Block[], name: string) {
  const PptxGenJS = (await import('pptxgenjs')).default
  const pptx = new PptxGenJS()
  let slide: ReturnType<typeof pptx.addSlide> | null = null
  let bullets: string[] = []
  const flush = () => {
    if (slide && bullets.length) {
      slide.addText(bullets.map((t) => ({ text: t, options: { bullet: true, fontSize: 16, breakLine: true } })), { x: 0.6, y: 1.4, w: 8.8, h: 5 })
    }
    bullets = []
  }
  for (const b of blocks) {
    if (b.type === 'h1' || b.type === 'h2') {
      flush()
      slide = pptx.addSlide()
      slide.addText(b.text ?? '', { x: 0.6, y: 0.4, w: 8.8, h: 0.9, fontSize: 26, bold: true })
    } else if (b.type === 'li' || b.type === 'p') {
      if (!slide) { slide = pptx.addSlide(); slide.addText(name, { x: 0.6, y: 0.4, w: 8.8, h: 0.9, fontSize: 26, bold: true }) }
      if (b.text) bullets.push(b.text)
    }
  }
  flush()
  saveBlob((await pptx.write({ outputType: 'blob' })) as Blob, name + '.pptx')
}

// ── Excel (.xlsx) ─────────────────────────────────────────────────────────────────────────────────
async function exportXlsx(blocks: Block[], name: string) {
  const ExcelJS = (await import('exceljs')).default
  const wb = new ExcelJS.Workbook()
  const ws = wb.addWorksheet('Sheet1')
  const table = blocks.find((b) => b.type === 'table' && b.rows?.length)
  if (table?.rows) {
    table.rows.forEach((r) => ws.addRow(r))
    ws.getRow(1).font = { bold: true }
  } else {
    blocks.forEach((b) => { if (b.text) ws.addRow([b.text]) })
  }
  ws.columns.forEach((c) => { c.width = 32 })
  const buf = await wb.xlsx.writeBuffer()
  saveBlob(new Blob([buf], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }), name + '.xlsx')
}

/** exportDoc renders the markdown into the chosen format and downloads it. */
export async function exportDoc(markdown: string, format: DocFormat, title: string): Promise<void> {
  const name = safeName(title)
  if (format === 'markdown') { saveBlob(new Blob([markdown], { type: 'text/markdown' }), name + '.md'); return }
  const blocks = parseMarkdown(markdown)
  if (format === 'pdf') return exportPDF(blocks, name)
  if (format === 'docx') return exportDocx(blocks, name)
  if (format === 'pptx') return exportPptx(blocks, name)
  if (format === 'xlsx') return exportXlsx(blocks, name)
}
