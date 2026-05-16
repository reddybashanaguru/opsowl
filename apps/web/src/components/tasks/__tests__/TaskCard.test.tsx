import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TaskCard, Task } from "../TaskCard";

function makeTask(overrides: Partial<Task> = {}): Task {
  return {
    id: "00000000-0000-0000-0000-000000000001",
    case_id: "00000000-0000-0000-0000-000000000002",
    title: "Verify PAN card",
    description: "Cross-check NSDL records",
    status: "todo",
    priority: "high",
    is_overdue: false,
    is_ai_suggested: false,
    due_date: null,
    completed_at: null,
    sort_order: 0,
    comment_count: 0,
    created_by: { id: "u1", full_name: "Alice Smith", email: "alice@test.com" },
    assigned_to: null,
    created_at: new Date(Date.now() - 3600_000).toISOString(),
    updated_at: new Date(Date.now() - 60_000).toISOString(),
    ...overrides,
  };
}

describe("TaskCard", () => {
  it("renders task title", () => {
    render(<TaskCard task={makeTask()} />);
    expect(screen.getByText("Verify PAN card")).toBeInTheDocument();
  });

  it("renders status and priority badges", () => {
    render(<TaskCard task={makeTask()} />);
    expect(screen.getByText("Todo")).toBeInTheDocument();
    expect(screen.getByText(/High/)).toBeInTheDocument();
  });

  it("shows overdue badge when is_overdue is true and not done", () => {
    render(<TaskCard task={makeTask({ is_overdue: true })} />);
    expect(screen.getByText("Overdue")).toBeInTheDocument();
  });

  it("does not show overdue badge when task is done", () => {
    render(<TaskCard task={makeTask({ is_overdue: true, status: "done" })} />);
    expect(screen.queryByText("Overdue")).not.toBeInTheDocument();
  });

  it("shows AI badge when is_ai_suggested is true", () => {
    render(<TaskCard task={makeTask({ is_ai_suggested: true })} />);
    expect(screen.getByText("AI")).toBeInTheDocument();
  });

  it("renders assignee name when assigned", () => {
    render(
      <TaskCard
        task={makeTask({
          assigned_to: { id: "u2", full_name: "Bob Jones", email: "bob@test.com" },
        })}
      />,
    );
    expect(screen.getByText(/Bob Jones/)).toBeInTheDocument();
  });

  it("renders due date", () => {
    const due = new Date("2026-06-01").toISOString();
    render(<TaskCard task={makeTask({ due_date: due })} />);
    expect(screen.getByText(/Due/)).toBeInTheDocument();
  });

  it("renders comment count when > 0", () => {
    render(<TaskCard task={makeTask({ comment_count: 5 })} />);
    expect(screen.getByText(/💬 5/)).toBeInTheDocument();
  });

  it("strikes through title when task is done", () => {
    render(<TaskCard task={makeTask({ status: "done" })} />);
    const title = screen.getByText("Verify PAN card");
    expect(title.className).toContain("line-through");
  });

  it("strikes through title when task is cancelled", () => {
    render(<TaskCard task={makeTask({ status: "cancelled" })} />);
    const title = screen.getByText("Verify PAN card");
    expect(title.className).toContain("line-through");
  });

  it("calls onClick when card is clicked", async () => {
    const onClick = vi.fn();
    render(<TaskCard task={makeTask()} onClick={onClick} />);
    await userEvent.click(screen.getByText("Verify PAN card"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("calls onComplete when checkbox is clicked and not done", async () => {
    const onComplete = vi.fn();
    render(<TaskCard task={makeTask()} onClick={vi.fn()} onComplete={onComplete} />);
    const checkbox = screen.getByTitle("Mark complete");
    await userEvent.click(checkbox);
    expect(onComplete).toHaveBeenCalledOnce();
  });

  it("does not call onComplete when task is already done", async () => {
    const onComplete = vi.fn();
    render(<TaskCard task={makeTask({ status: "done" })} onComplete={onComplete} />);
    const checkbox = screen.getByTitle("Completed");
    await userEvent.click(checkbox);
    expect(onComplete).not.toHaveBeenCalled();
  });

  it("shows case_number when showCase is true", () => {
    render(
      <TaskCard
        task={makeTask({ case_number: "OWL-0042" })}
        showCase
      />,
    );
    expect(screen.getByText("OWL-0042")).toBeInTheDocument();
  });

  it("does not show case_number when showCase is false", () => {
    render(
      <TaskCard
        task={makeTask({ case_number: "OWL-0042" })}
        showCase={false}
      />,
    );
    expect(screen.queryByText("OWL-0042")).not.toBeInTheDocument();
  });

  it("truncates long description with line-clamp class", () => {
    render(<TaskCard task={makeTask({ description: "A ".repeat(200) })} />);
    const desc = screen.getByText(/A A A/);
    expect(desc.className).toContain("line-clamp");
  });
});
