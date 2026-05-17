import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { DocumentUploadZone } from "../DocumentUploadZone";

function makePDFFile(name = "test.pdf", size = 1024) {
  return new File(["a".repeat(size)], name, { type: "application/pdf" });
}

describe("DocumentUploadZone", () => {
  it("renders upload prompt", () => {
    render(<DocumentUploadZone onUpload={vi.fn()} />);
    expect(screen.getByText(/Drop a file here/)).toBeInTheDocument();
  });

  it("shows uploading state", () => {
    render(<DocumentUploadZone onUpload={vi.fn()} uploading />);
    expect(screen.getByText("Uploading…")).toBeInTheDocument();
  });

  it("calls onUpload with selected file", async () => {
    const onUpload = vi.fn().mockResolvedValue(undefined);
    render(<DocumentUploadZone onUpload={onUpload} />);

    const input = screen.getByTestId("file-input");
    const file = makePDFFile();
    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => expect(onUpload).toHaveBeenCalledWith(file));
  });

  it("shows error when onUpload rejects", async () => {
    const onUpload = vi.fn().mockRejectedValue(new Error("Server error"));
    render(<DocumentUploadZone onUpload={onUpload} />);

    const input = screen.getByTestId("file-input");
    fireEvent.change(input, { target: { files: [makePDFFile()] } });

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Server error"));
  });

  it("shows error for oversized file without calling onUpload", async () => {
    const onUpload = vi.fn();
    render(<DocumentUploadZone onUpload={onUpload} />);

    const bigFile = new File(["x".repeat(100)], "huge.pdf", { type: "application/pdf" });
    Object.defineProperty(bigFile, "size", { value: 51 * 1024 * 1024 });

    const input = screen.getByTestId("file-input");
    fireEvent.change(input, { target: { files: [bigFile] } });

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(/50 MB limit/),
    );
    expect(onUpload).not.toHaveBeenCalled();
  });

  it("handles drag-over state", () => {
    const { container } = render(<DocumentUploadZone onUpload={vi.fn()} />);
    const zone = container.querySelector("[role=button]")!;
    fireEvent.dragOver(zone);
    expect(zone.className).toContain("border-cyan-400");
  });

  it("clears drag-over on drag leave", () => {
    const { container } = render(<DocumentUploadZone onUpload={vi.fn()} />);
    const zone = container.querySelector("[role=button]")!;
    fireEvent.dragOver(zone);
    fireEvent.dragLeave(zone);
    expect(zone.className).not.toContain("border-cyan-400");
  });

  it("calls onUpload on file drop", async () => {
    const onUpload = vi.fn().mockResolvedValue(undefined);
    const { container } = render(<DocumentUploadZone onUpload={onUpload} />);
    const zone = container.querySelector("[role=button]")!;

    const file = makePDFFile("dropped.pdf");
    fireEvent.drop(zone, { dataTransfer: { files: [file] } });

    await waitFor(() => expect(onUpload).toHaveBeenCalledWith(file));
  });
});
