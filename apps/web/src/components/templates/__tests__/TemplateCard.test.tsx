import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TemplateCard, Template } from "../TemplateCard";

function makeTemplate(overrides: Partial<Template> = {}): Template {
  return {
    id: "00000000-0000-0000-0000-000000000001",
    name: "UPI Fraud Template",
    description_skeleton: "Investigate suspicious UPI transaction cluster.",
    priority: "high",
    tags: ["upi", "fraud"],
    is_active: true,
    tasks: [
      { title: "Verify transaction", description: "", priority: "high", sort_order: 0 },
      { title: "Contact bank", description: "", priority: "medium", sort_order: 1 },
    ],
    created_at: "2026-05-01T00:00:00Z",
    updated_at: "2026-05-20T00:00:00Z",
    ...overrides,
  };
}

describe("TemplateCard", () => {
  it("renders the template name", () => {
    render(<TemplateCard template={makeTemplate()} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.getByText("UPI Fraud Template")).toBeInTheDocument();
  });

  it("shows active badge when is_active is true", () => {
    render(<TemplateCard template={makeTemplate({ is_active: true })} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.getByTestId("active-badge")).toHaveTextContent("active");
  });

  it("shows inactive badge when is_active is false", () => {
    render(<TemplateCard template={makeTemplate({ is_active: false })} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.getByTestId("active-badge")).toHaveTextContent("inactive");
  });

  it("shows the priority badge", () => {
    render(<TemplateCard template={makeTemplate({ priority: "critical" })} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.getByTestId("priority-badge")).toHaveTextContent("critical");
  });

  it("renders description skeleton", () => {
    render(<TemplateCard template={makeTemplate()} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.getByText("Investigate suspicious UPI transaction cluster.")).toBeInTheDocument();
  });

  it("renders tags", () => {
    render(<TemplateCard template={makeTemplate()} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.getByText("upi")).toBeInTheDocument();
    expect(screen.getByText("fraud")).toBeInTheDocument();
  });

  it("does not render tags section when tags are empty", () => {
    render(<TemplateCard template={makeTemplate({ tags: [] })} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.queryByText("upi")).not.toBeInTheDocument();
  });

  it("shows task count", () => {
    render(<TemplateCard template={makeTemplate()} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.getByTestId("task-count")).toHaveTextContent("2 tasks in checklist");
  });

  it("shows singular 'task' for 1 task", () => {
    render(
      <TemplateCard
        template={makeTemplate({ tasks: [{ title: "Only task", description: "", priority: "medium", sort_order: 0 }] })}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByTestId("task-count")).toHaveTextContent("1 task in checklist");
  });

  it("renders first 3 task titles", () => {
    render(<TemplateCard template={makeTemplate()} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.getByText("Verify transaction")).toBeInTheDocument();
    expect(screen.getByText("Contact bank")).toBeInTheDocument();
  });

  it("shows overflow count when more than 3 tasks", () => {
    const template = makeTemplate({
      tasks: [
        { title: "Task 1", description: "", priority: "medium", sort_order: 0 },
        { title: "Task 2", description: "", priority: "medium", sort_order: 1 },
        { title: "Task 3", description: "", priority: "medium", sort_order: 2 },
        { title: "Task 4", description: "", priority: "medium", sort_order: 3 },
        { title: "Task 5", description: "", priority: "medium", sort_order: 4 },
      ],
    });
    render(<TemplateCard template={template} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.getByTestId("overflow-count")).toHaveTextContent("+2 more…");
  });

  it("does not show overflow count when tasks ≤ 3", () => {
    render(<TemplateCard template={makeTemplate()} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.queryByTestId("overflow-count")).not.toBeInTheDocument();
  });

  it("calls onEdit with the template when Edit is clicked", async () => {
    const onEdit = vi.fn();
    render(<TemplateCard template={makeTemplate()} onEdit={onEdit} onDelete={vi.fn()} />);
    await userEvent.click(screen.getByText("Edit"));
    expect(onEdit).toHaveBeenCalledWith(expect.objectContaining({ id: "00000000-0000-0000-0000-000000000001" }));
  });

  it("calls onDelete with the template ID when Delete is clicked", async () => {
    const onDelete = vi.fn();
    render(<TemplateCard template={makeTemplate()} onEdit={vi.fn()} onDelete={onDelete} />);
    await userEvent.click(screen.getByText("Delete"));
    expect(onDelete).toHaveBeenCalledWith("00000000-0000-0000-0000-000000000001");
  });

  it("shows 0 tasks in checklist when tasks array is empty", () => {
    render(<TemplateCard template={makeTemplate({ tasks: [] })} onEdit={vi.fn()} onDelete={vi.fn()} />);
    expect(screen.getByTestId("task-count")).toHaveTextContent("0 tasks in checklist");
  });
});
