export type ParsedSSEEvent = {
  event: string;
  data: unknown;
  rawData: string;
  malformed: boolean;
};

export type ParseSSEResult = {
  events: ParsedSSEEvent[];
  remainder: string;
};

export function parseSSEChunk(buffer: string, chunk: string): ParseSSEResult {
  return parseSSEBuffer(`${buffer}${chunk}`);
}

export function parseSSEBuffer(buffer: string): ParseSSEResult {
  const normalized = buffer.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
  const blocks = normalized.split("\n\n");
  const remainder = blocks.pop() ?? "";
  const events = blocks.map(parseSSEBlock).filter((event): event is ParsedSSEEvent => event !== null);

  return { events, remainder };
}

function parseSSEBlock(block: string): ParsedSSEEvent | null {
  const lines = block.split("\n");
  let eventName = "message";
  const dataLines: string[] = [];

  for (const line of lines) {
    if (!line || line.startsWith(":")) {
      continue;
    }
    const separator = line.indexOf(":");
    const field = separator === -1 ? line : line.slice(0, separator);
    let value = separator === -1 ? "" : line.slice(separator + 1);
    if (value.startsWith(" ")) {
      value = value.slice(1);
    }

    if (field === "event") {
      eventName = value || "message";
    } else if (field === "data") {
      dataLines.push(value);
    }
  }

  if (dataLines.length === 0 && eventName === "message") {
    return null;
  }

  const rawData = dataLines.join("\n");
  if (!rawData) {
    return { event: eventName, data: null, rawData, malformed: false };
  }

  try {
    return { event: eventName, data: JSON.parse(rawData), rawData, malformed: false };
  } catch {
    return { event: eventName, data: null, rawData, malformed: true };
  }
}
