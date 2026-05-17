import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { CitationCard, Citation } from "../CitationCard";

function makeCitation(overrides: Partial<Citation> = {}): Citation {
  return {
    doc_id: "doc-001",
    doc_name: "evidence.pdf",
    chunk_index: 0,
    page_number: 3,
    excerpt: "The account showed three suspicious transfers in March 2026.",
    similarity: 0.87,
    ...overrides,
  };
}

describe("CitationCard", () => {
  it("renders source label with 1-based index", () => {
    render(<CitationCard citation={makeCitation()} index={0} />);
    expect(screen.getByText("Source 1")).toBeInTheDocument();
  });

  it("renders second source as Source 2", () => {
    render(<CitationCard citation={makeCitation()} index={1} />);
    expect(screen.getByText("Source 2")).toBeInTheDocument();
  });

  it("renders document name", () => {
    render(<CitationCard citation={makeCitation()} index={0} />);
    expect(screen.getByText("evidence.pdf")).toBeInTheDocument();
  });

  it("renders page number when present", () => {
    render(<CitationCard citation={makeCitation({ page_number: 5 })} index={0} />);
    expect(screen.getByText("Page 5")).toBeInTheDocument();
  });

  it("omits page number when null", () => {
    render(<CitationCard citation={makeCitation({ page_number: null })} index={0} />);
    expect(screen.queryByText(/Page/)).not.toBeInTheDocument();
  });

  it("renders similarity percentage", () => {
    render(<CitationCard citation={makeCitation({ similarity: 0.87 })} index={0} />);
    expect(screen.getByText("87% match")).toBeInTheDocument();
  });

  it("renders excerpt text", () => {
    render(<CitationCard citation={makeCitation()} index={0} />);
    expect(screen.getByText(/"The account showed/)).toBeInTheDocument();
  });

  it("has testid citation-card", () => {
    render(<CitationCard citation={makeCitation()} index={0} />);
    expect(screen.getByTestId("citation-card")).toBeInTheDocument();
  });
});
