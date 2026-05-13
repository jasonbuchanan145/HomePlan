import type { HouseState, Room, Task, TaskStatus, TaskWithContext } from "../types/house";

export type SortDirection = "asc" | "desc";
export type SortKey = "title" | "area" | "priority" | "type" | "status" | "percentComplete" | "dateStarted" | "completedOn";

export function getRoomTasks(house: HouseState, room: Room): Task[] {
  return room.sharedTaskSet ? house.roomTaskSets[room.sharedTaskSet] ?? [] : room.tasks ?? [];
}

export function allTasks(house: HouseState): TaskWithContext[] {
  const roomTasks = Object.entries(house.floors).flatMap(([floorKey, floor]) =>
    floor.rooms.flatMap((room) =>
      getRoomTasks(house, room).map((task) => ({
        ...task,
        area: room.name,
        floor: floor.label,
        floorKey,
        roomId: room.id,
        groupKey: ""
      }))
    )
  );

  const groupTasks = Object.entries(house.taskGroups).flatMap(([groupKey, group]) =>
    group.tasks.map((task) => ({
      ...task,
      area: group.label,
      floor: "Project List",
      floorKey: "",
      roomId: "",
      groupKey
    }))
  );

  const uniqueTasks = new Map<string, TaskWithContext>();
  [...roomTasks, ...groupTasks].forEach((task) => {
    if (!uniqueTasks.has(task.id)) {
      uniqueTasks.set(task.id, task);
      return;
    }

    if (task.id.startsWith("stairs-")) {
      const existing = uniqueTasks.get(task.id);
      if (existing) uniqueTasks.set(task.id, { ...existing, area: "Stairs", floor: "Top Floor / Bottom Floor" });
    }
  });

  return [...uniqueTasks.values()];
}

export function formatDate(dateValue?: string): string {
  if (!dateValue) return "-";
  const [year, month, day] = dateValue.split("-");
  return `${Number(month)}/${Number(day)}/${year}`;
}

export function statusLabel(task: { status: TaskStatus }): string {
  if (task.status === "done") return "Done";
  if (task.status === "in-progress") return "In Progress";
  if (task.status === "blocked") return "Blocked";
  return "Open";
}

export function taskPercent(task: Task): number {
  if (Number.isFinite(task.percentComplete)) return task.percentComplete ?? 0;
  return task.status === "done" ? 100 : 0;
}

export function badgeClass(task: Task): string {
  if (task.status === "done") return "done";
  if (task.status === "blocked") return "blocked";
  if (task.status === "in-progress") return "in-progress";
  if (task.priority === "critical") return "urgent";
  if (task.priority === "later") return "later";
  if (task.type === "contractor") return "contractor";
  return "diy";
}

export function badgeLabel(task: Task): string {
  if (task.status === "done") return "done";
  if (task.status === "blocked") return "blocked";
  if (task.status === "in-progress") return "in progress";
  if (task.priority === "critical") return "urgent";
  if (task.priority === "later") return "later";
  if (task.type === "contractor") return "contractor";
  return "DIY";
}

export function taskBadges(task: Task): Array<{ className: string; label: string }> {
  const badges = [{ className: badgeClass(task), label: badgeLabel(task) }];
  if (task.status !== "done" && task.type === "contractor" && badgeClass(task) !== "contractor") {
    badges.push({ className: "contractor", label: "contractor" });
  }
  if (task.status !== "done" && task.type === "DIY" && badgeClass(task) !== "diy") {
    badges.push({ className: "diy", label: "DIY" });
  }
  return badges;
}

export function roomBadges(house: HouseState, room: Room): string[] {
  const tasks = getRoomTasks(house, room);
  const badges: string[] = [];
  if (tasks.some((task) => task.priority === "critical" && task.status !== "done")) badges.push("urgent");
  if (tasks.some((task) => task.type === "contractor" && task.status !== "done")) badges.push("contractor");
  if (tasks.length && tasks.every((task) => task.status === "done")) badges.push("done");
  return badges;
}

function sortValue(task: TaskWithContext, key: SortKey): string | number {
  const priorityRank: Record<string, number> = { critical: 1, important: 2, later: 3, complete: 4 };
  const statusRank: Record<string, number> = { "in-progress": 1, blocked: 2, open: 3, done: 4 };
  const typeRank: Record<string, number> = { contractor: 1, DIY: 2, done: 3 };

  if (key === "priority") return priorityRank[task.priority] || 99;
  if (key === "status") return statusRank[task.status] || 99;
  if (key === "type") return typeRank[task.type] || 99;
  if (key === "percentComplete") return taskPercent(task);
  if (key === "dateStarted") return task.dateStarted || "9999-12-31";
  if (key === "completedOn") return task.completedOn || "9999-12-31";
  return String(task[key] || "").toLowerCase();
}

export function sortedTasks(tasks: TaskWithContext[], key: SortKey, direction: SortDirection): TaskWithContext[] {
  const multiplier = direction === "asc" ? 1 : -1;
  return [...tasks].sort((a, b) => {
    const aValue = sortValue(a, key);
    const bValue = sortValue(b, key);
    if (aValue < bValue) return -1 * multiplier;
    if (aValue > bValue) return 1 * multiplier;
    return a.title.localeCompare(b.title);
  });
}
