const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('electronAPI', {
  // Platform info
  platform: process.platform,

  // Window controls (used by Linux titlebar)
  minimize: () => ipcRenderer.invoke('window:minimize'),
  maximize: () => ipcRenderer.invoke('window:maximize'),
  close: () => ipcRenderer.invoke('window:close'),
  isMaximized: () => ipcRenderer.invoke('window:isMaximized'),

  // Listen for maximize state changes
  onMaximizeChange: (callback: (maximized: boolean) => void) => {
    ipcRenderer.on('window:maximize-changed', (_event: any, maximized: boolean) => callback(maximized))
  },

  // Menu events from native menu bar
  onMenuAction: (callback: (action: string) => void) => {
    ipcRenderer.on('menu:new-card', () => callback('new-card'))
    ipcRenderer.on('menu:toggle-sidebar', () => callback('toggle-sidebar'))
  },
})
