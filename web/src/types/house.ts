export type TaskStatus = "open" | "in-progress" | "done" | "blocked";
export type TaskPriority = "critical" | "important" | "later";
export type TaskType = "DIY" | "contractor" | "done";

export interface RoomLayout {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface ItemNeeded {
  name: string;
  status?: string;
}

export interface Subtask {
  id: string;
  title: string;
  status: TaskStatus;
  dateStarted?: string;
  completedOn?: string;
  percentComplete?: number;
}

export interface Task {
  id: string;
  title: string;
  priority: TaskPriority;
  type: TaskType;
  status: TaskStatus;
  notes?: string;
  dateStarted?: string;
  completedOn?: string;
  percentComplete?: number;
  doFirst?: boolean;
  doFirstRank?: number;
  subtasks?: Subtask[];
  itemsNeeded?: ItemNeeded[];
}

export interface RoomDisplay {
  compact?: boolean;
  mapLabel?: string;
  borderStyle?: "solid" | "dashed";
  tone?: "default" | "outdoor";
}

export interface Room {
  id: string;
  name: string;
  layout: RoomLayout;
  display?: RoomDisplay;
  tasks?: Task[];
  sharedTaskSet?: string;
}

export interface Floor {
  label: string;
  defaultRoom: string;
  grid: {
    columns: number;
    rows: number;
  };
  rooms: Room[];
}

export interface TaskGroup {
  label: string;
  description: string;
  tasks: Task[];
}

export interface HouseState {
  schemaVersion: 1;
  id: string;
  name: string;
  taskGroups: Record<string, TaskGroup>;
  roomTaskSets: Record<string, Task[]>;
  floors: Record<string, Floor>;
  unplacedRooms?: Room[];
}

export interface TaskWithContext extends Task {
  area: string;
  floor: string;
  floorKey: string;
  roomId: string;
  groupKey: string;
  isUnplaced?: boolean;
}

export interface TaskTargetOption {
  value: string;
  label: string;
  needsRoomName?: boolean;
}

export interface TaskCreateDraft {
  title: string;
  target: string;
  priority: TaskPriority;
  type: TaskType;
  status: TaskStatus;
  notes?: string;
  roomName?: string;
  subtasks: Array<Pick<Subtask, "title" | "status">>;
  itemsNeeded: Array<Pick<ItemNeeded, "name" | "status">>;
}
