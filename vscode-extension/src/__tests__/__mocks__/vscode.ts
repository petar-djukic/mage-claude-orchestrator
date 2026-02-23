// Minimal vscode module stub for unit tests.

export class EventEmitter {
  event = () => {};
  fire() {}
  dispose() {}
}

export enum TreeItemCollapsibleState {
  None = 0,
  Collapsed = 1,
  Expanded = 2,
}

export class TreeItem {
  label: string;
  collapsibleState: TreeItemCollapsibleState;
  description?: string;
  contextValue?: string;
  iconPath?: unknown;
  tooltip?: string;
  command?: unknown;

  constructor(label: string, collapsibleState?: TreeItemCollapsibleState) {
    this.label = label;
    this.collapsibleState = collapsibleState ?? TreeItemCollapsibleState.None;
  }
}

export class ThemeIcon {
  id: string;
  constructor(id: string) {
    this.id = id;
  }
}

export class Range {
  constructor(
    public startLine: number,
    public startChar: number,
    public endLine: number,
    public endChar: number
  ) {}
}

export class CodeLens {
  range: Range;
  command?: unknown;
  constructor(range: Range, command?: unknown) {
    this.range = range;
    this.command = command;
  }
}

export class Uri {
  static file(path: string) {
    return { scheme: "file", path };
  }
}

export const window = {
  showTextDocument: async () => {},
  showWarningMessage: () => {},
};
