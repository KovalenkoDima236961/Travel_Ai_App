import type { ExportTrip } from "@/lib/export/trip-export-adapter";
import { buildTripPdfLines } from "@/lib/export/trip-pdf-lines";

export {
  buildTripPdfLines,
  type TripPdfLine,
  type TripPdfLineVariant
} from "@/lib/export/trip-pdf-lines";

export function TripPdfDocument({ exportTrip }: { exportTrip: ExportTrip }) {
  return (
    <article>
      {buildTripPdfLines(exportTrip).map((line, index) => {
        const Tag = line.variant === "title" ? "h1" : line.variant === "heading" ? "h2" : "p";
        return (
          <Tag key={`${line.variant ?? "body"}-${index}`} style={{ marginLeft: line.indent ?? 0 }}>
            {line.text}
          </Tag>
        );
      })}
    </article>
  );
}
