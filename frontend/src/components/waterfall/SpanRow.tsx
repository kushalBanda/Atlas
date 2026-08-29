import type { SpanNode } from "../../lib/spanTree";

const CONNECTOR_WIDTH = 20;

export interface SpanRowProps {
  node: SpanNode;
  selected: boolean;
  isRootCause: boolean;
  dim: boolean;
  onClick: (spanId: string) => void;
}

// Nesting via absolute-positioned indentation, not text padding tricks,
// matching SigNoz's tree model (see design doc 4.2).
export function SpanRow({ node, selected, isRootCause, dim, onClick }: SpanRowProps) {
  const { span, depth, children } = node;
  const isError = span.StatusCode === "error";

  return (
    <div
      data-span-id={span.SpanID}
      className={`flex h-[30px] cursor-pointer items-center border-b border-border text-xs hover:bg-elevated ${
        selected ? "bg-accent-dim" : isError ? "bg-error-row" : ""
      } ${dim ? "opacity-40" : ""}`}
      style={{ paddingLeft: 12 + depth * CONNECTOR_WIDTH }}
      onClick={() => onClick(span.SpanID)}
    >
      <span
        className={`mr-1.5 h-[7px] w-[7px] flex-shrink-0 rounded-full ${
          isError ? "bg-error" : "bg-[#6ba3d6]"
        } ${isRootCause ? "ring-1 ring-accent" : ""}`}
      />
      <span className="overflow-hidden text-ellipsis whitespace-nowrap">
        {span.ServiceName} &middot; {span.Name}
      </span>
      {children.length > 0 && (
        <span className="ml-1.5 flex-shrink-0 text-[10px] text-text-faint">{children.length}</span>
      )}
    </div>
  );
}
