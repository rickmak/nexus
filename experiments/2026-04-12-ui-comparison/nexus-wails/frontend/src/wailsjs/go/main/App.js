// @ts-check
export function GetWorkspaces() {
  return window['go']['main']['App']['GetWorkspaces']();
}
export function WorkspaceAction(id, action) {
  return window['go']['main']['App']['WorkspaceAction'](id, action);
}
