import type { HouseState, Room, Task, TaskPriority, TaskStatus, TaskType, TaskWithContext } from "../types/house";

export type SortDirection = "asc" | "desc";
export type SortKey = "title" | "area" | "priority" | "type" | "status" | "percentComplete" | "dateStarted" | "completedOn";

export const TASK_STATUSES: TaskStatus[] = ["open", "in-progress", "blocked", "done"];
export const TASK_PRIORITIES: TaskPriority[] = ["critical", "important", "later", "complete"];
export const TASK_TYPES: TaskType[] = ["DIY", "contractor", "done"];

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

  const unplacedTasks = (house.unplacedRooms ?? []).flatMap((room) =>
    (room.tasks ?? []).map((task) => ({
      ...task,
      area: room.name,
      floor: "Unplaced",
      floorKey: "",
      roomId: room.id,
      groupKey: "",
      isUnplaced: true
    }))
  );

  const uniqueTasks = new Map<string, TaskWithContext>();
  [...roomTasks, ...groupTasks, ...unplacedTasks].forEach((task) => {
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
  if (task.status !== "in-progress") return 0;
  if (Number.isFinite(task.percentComplete)) return task.percentComplete ?? 0;
  return 0;
}

export function clampTaskPercent(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(100, Math.round(value)));
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

export function updateHouseTask(house: HouseState, taskId: string, updateTask: (task: Task) => Task): HouseState {
  const updateTasks = (taskList?: Task[]): Task[] | undefined => {
    if (!taskList) return taskList;
    let changed = false;
    const nextTasks = taskList.map((task) => {
      if (task.id !== taskId) return task;
      changed = true;
      return updateTask(task);
    });
    return changed ? nextTasks : taskList;
  };

  return {
    ...house,
    taskGroups: Object.fromEntries(
      Object.entries(house.taskGroups).map(([groupKey, group]) => [
        groupKey,
        {
          ...group,
          tasks: updateTasks(group.tasks) ?? []
        }
      ])
    ),
    roomTaskSets: Object.fromEntries(
      Object.entries(house.roomTaskSets).map(([setKey, taskList]) => [setKey, updateTasks(taskList) ?? []])
    ),
    floors: Object.fromEntries(
      Object.entries(house.floors).map(([floorKey, floor]) => [
        floorKey,
        {
          ...floor,
          rooms: floor.rooms.map((room) => ({
            ...room,
            tasks: updateTasks(room.tasks)
          }))
        }
      ])
    ),
    unplacedRooms: house.unplacedRooms?.map((room) => ({
      ...room,
      tasks: updateTasks(room.tasks)
    }))
  };
}

export function deleteHouseTask(house: HouseState, taskId: string): HouseState {
  const deleteTasks = (taskList?: Task[]): Task[] | undefined => taskList?.filter((task) => task.id !== taskId);

  return {
    ...house,
    taskGroups: Object.fromEntries(
      Object.entries(house.taskGroups).map(([groupKey, group]) => [
        groupKey,
        {
          ...group,
          tasks: deleteTasks(group.tasks) ?? []
        }
      ])
    ),
    roomTaskSets: Object.fromEntries(
      Object.entries(house.roomTaskSets).map(([setKey, taskList]) => [setKey, deleteTasks(taskList) ?? []])
    ),
    floors: Object.fromEntries(
      Object.entries(house.floors).map(([floorKey, floor]) => [
        floorKey,
        {
          ...floor,
          rooms: floor.rooms.map((room) => ({
            ...room,
            tasks: deleteTasks(room.tasks)
          }))
        }
      ])
    ),
    unplacedRooms: house.unplacedRooms?.map((room) => ({
      ...room,
      tasks: deleteTasks(room.tasks)
    }))
  };
}
