import { buildTripPdfLines, type TripPdfLine, type TripPdfLineVariant } from "@/components/export/TripPdfDocument";
import { downloadBlob } from "@/lib/export/download";
import { buildPdfFilename } from "@/lib/export/export-filenames";
import type { ExportTrip } from "@/lib/export/trip-export-adapter";

const PAGE_WIDTH = 595.28;
const PAGE_HEIGHT = 841.89;
const MARGIN_X = 48;
const MARGIN_TOP = 56;
const MARGIN_BOTTOM = 52;

type RenderedLine = {
  text: string;
  x: number;
  y: number;
  variant: TripPdfLineVariant;
};

export async function downloadTripPdf(exportTrip: ExportTrip): Promise<void> {
  const blob = createTripPdfBlob(exportTrip);
  downloadBlob(blob, buildPdfFilename(exportTrip));
}

export function createTripPdfBlob(exportTrip: ExportTrip): Blob {
  const pages = paginateLines(buildTripPdfLines(exportTrip));
  const pdf = buildPdfDocument(pages);
  return new Blob([pdf], { type: "application/pdf" });
}

function paginateLines(lines: TripPdfLine[]): RenderedLine[][] {
  const pages: RenderedLine[][] = [[]];
  let y = PAGE_HEIGHT - MARGIN_TOP;

  for (const line of lines) {
    const variant = line.variant ?? "body";
    const style = lineStyle(variant);
    const indent = line.indent ?? 0;
    const wrappedLines = wrapText(line.text, maxCharsForLine(style.fontSize, indent));
    const topSpacing = style.marginTop;

    if (y - topSpacing - style.lineHeight < MARGIN_BOTTOM) {
      pages.push([]);
      y = PAGE_HEIGHT - MARGIN_TOP;
    } else {
      y -= topSpacing;
    }

    for (const wrappedLine of wrappedLines) {
      if (y - style.lineHeight < MARGIN_BOTTOM) {
        pages.push([]);
        y = PAGE_HEIGHT - MARGIN_TOP;
      }

      pages[pages.length - 1].push({
        text: wrappedLine,
        x: MARGIN_X + indent,
        y,
        variant
      });
      y -= style.lineHeight;
    }
  }

  return pages.filter((page) => page.length > 0);
}

function buildPdfDocument(pages: RenderedLine[][]): string {
  const encoder = new TextEncoder();
  const objects: string[] = [];

  objects[0] = "<< /Type /Catalog /Pages 2 0 R >>";
  objects[2] = "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>";
  objects[3] = "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>";
  objects[4] = "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Oblique >>";

  const pageRefs: string[] = [];
  pages.forEach((page, index) => {
    const pageObjectId = 6 + index * 2;
    const contentObjectId = pageObjectId + 1;
    const content = buildPageContent(page);

    pageRefs.push(`${pageObjectId} 0 R`);
    objects[pageObjectId - 1] =
      `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ${PAGE_WIDTH} ${PAGE_HEIGHT}] /Resources << /Font << /F1 3 0 R /F2 4 0 R /F3 5 0 R >> >> /Contents ${contentObjectId} 0 R >>`;
    objects[contentObjectId - 1] =
      `<< /Length ${encoder.encode(content).length} >>\nstream\n${content}\nendstream`;
  });

  objects[1] = `<< /Type /Pages /Kids [${pageRefs.join(" ")}] /Count ${pages.length} >>`;

  let pdf = "%PDF-1.4\n";
  const offsets = [0];

  objects.forEach((object, index) => {
    offsets.push(encoder.encode(pdf).length);
    pdf += `${index + 1} 0 obj\n${object}\nendobj\n`;
  });

  const xrefOffset = encoder.encode(pdf).length;
  pdf += `xref\n0 ${objects.length + 1}\n`;
  pdf += "0000000000 65535 f \n";
  for (const offset of offsets.slice(1)) {
    pdf += `${String(offset).padStart(10, "0")} 00000 n \n`;
  }
  pdf += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xrefOffset}\n%%EOF\n`;

  return pdf;
}

function buildPageContent(page: RenderedLine[]): string {
  return page
    .map((line) => {
      const style = lineStyle(line.variant);
      return [
        "BT",
        `${style.color} rg`,
        `/${style.font} ${style.fontSize} Tf`,
        `1 0 0 1 ${line.x.toFixed(2)} ${line.y.toFixed(2)} Tm`,
        `(${escapePdfText(line.text)}) Tj`,
        "ET"
      ].join("\n");
    })
    .join("\n");
}

function wrapText(text: string, maxChars: number): string[] {
  const normalized = text.replace(/\s+/g, " ").trim();
  if (!normalized) {
    return [""];
  }

  const lines: string[] = [];
  let current = "";

  for (const word of normalized.split(" ")) {
    if (word.length > maxChars) {
      if (current) {
        lines.push(current);
        current = "";
      }
      for (let index = 0; index < word.length; index += maxChars) {
        lines.push(word.slice(index, index + maxChars));
      }
      continue;
    }

    const next = current ? `${current} ${word}` : word;
    if (next.length > maxChars && current) {
      lines.push(current);
      current = word;
    } else {
      current = next;
    }
  }

  if (current) {
    lines.push(current);
  }

  return lines;
}

function maxCharsForLine(fontSize: number, indent: number): number {
  const usableWidth = PAGE_WIDTH - MARGIN_X * 2 - indent;
  return Math.max(24, Math.floor(usableWidth / (fontSize * 0.5)));
}

function lineStyle(variant: TripPdfLineVariant) {
  switch (variant) {
    case "title":
      return { font: "F2", fontSize: 24, lineHeight: 30, marginTop: 0, color: "0.05 0.07 0.12" };
    case "subtitle":
      return { font: "F1", fontSize: 12, lineHeight: 18, marginTop: 4, color: "0.28 0.33 0.41" };
    case "heading":
      return { font: "F2", fontSize: 16, lineHeight: 22, marginTop: 18, color: "0.08 0.10 0.15" };
    case "subheading":
      return { font: "F2", fontSize: 13, lineHeight: 19, marginTop: 14, color: "0.10 0.12 0.18" };
    case "small":
      return { font: "F1", fontSize: 9, lineHeight: 13, marginTop: 2, color: "0.36 0.41 0.49" };
    case "muted":
      return { font: "F1", fontSize: 10, lineHeight: 15, marginTop: 3, color: "0.32 0.36 0.43" };
    case "footer":
      return { font: "F3", fontSize: 9, lineHeight: 13, marginTop: 20, color: "0.42 0.46 0.54" };
    default:
      return { font: "F1", fontSize: 11, lineHeight: 16, marginTop: 4, color: "0.13 0.16 0.22" };
  }
}

function escapePdfText(value: string): string {
  return value
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^\x09\x0a\x0d\x20-\x7e]/g, "?")
    .replace(/\\/g, "\\\\")
    .replace(/\(/g, "\\(")
    .replace(/\)/g, "\\)");
}
