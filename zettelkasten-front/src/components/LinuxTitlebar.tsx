import { useEffect, useState } from 'react'

const isElectron = typeof window !== 'undefined' && window.electronAPI
const isLinux = isElectron && window.electronAPI!.platform === 'linux'

const TITLEBAR_HEIGHT = 38

export function LinuxTitlebar() {
  if (!isLinux) return null

  return (
    <>
      <DragRegion />
      <WindowControls />
    </>
  )
}

function DragRegion() {
  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        height: TITLEBAR_HEIGHT,
        zIndex: 9999,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        // @ts-expect-error — Electron supports -webkit-app-region for drag
        WebkitAppRegion: 'drag',
        userSelect: 'none',
        background: '#FFFFFF',
        borderBottom: '1px solid #E5E5E5',
      }}
    >
      <span
        style={{
          fontSize: 13,
          fontWeight: 500,
          color: '#37352F',
        }}
      >
        Zettelgarden
      </span>
    </div>
  )
}

function WindowControls() {
  const [maximized, setMaximized] = useState(false)
  const api = window.electronAPI!

  useEffect(() => {
    api.isMaximized().then(setMaximized).catch(() => {})
    const onResize = () => {
      api.isMaximized().then(setMaximized).catch(() => {})
    }
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [api])

  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        right: 0,
        height: TITLEBAR_HEIGHT,
        zIndex: 10000,
        display: 'flex',
        alignItems: 'center',
        // @ts-expect-error — Electron supports -webkit-app-region for no-drag
        WebkitAppRegion: 'no-drag',
      }}
    >
      <TitlebarButton ariaLabel="Minimize" onClick={() => api.minimize()}>
        <MinimizeIcon />
      </TitlebarButton>
      <TitlebarButton
        ariaLabel={maximized ? 'Restore' : 'Maximize'}
        onClick={() => api.maximize().then(() => api.isMaximized().then(setMaximized))}
      >
        {maximized ? <RestoreIcon /> : <MaximizeIcon />}
      </TitlebarButton>
      <TitlebarButton ariaLabel="Close" close onClick={() => api.close()}>
        <CloseIcon />
      </TitlebarButton>
    </div>
  )
}

function TitlebarButton({
  ariaLabel,
  children,
  close,
  onClick,
}: {
  ariaLabel: string
  children: React.ReactNode
  close?: boolean
  onClick: () => void
}) {
  const [hovered, setHovered] = useState(false)

  return (
    <button
      aria-label={ariaLabel}
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        width: 46,
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        border: 'none',
        cursor: 'pointer',
        background: close && hovered ? '#E81123' : hovered ? 'rgba(0,0,0,0.06)' : 'transparent',
        color: close && hovered ? '#FFF' : '#37352F',
      }}
    >
      {children}
    </button>
  )
}

function MinimizeIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round">
      <line x1="2.5" y1="6" x2="9.5" y2="6" />
    </svg>
  )
}

function MaximizeIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.2">
      <rect x="2.5" y="2.5" width="7" height="7" rx="0.5" />
    </svg>
  )
}

function RestoreIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.2">
      <rect x="2.5" y="3.8" width="6" height="6" rx="0.5" />
      <path d="M4 3.8 V 2.5 H 9.5 V 8" />
    </svg>
  )
}

function CloseIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round">
      <line x1="3" y1="3" x2="9" y2="9" />
      <line x1="9" y1="3" x2="3" y2="9" />
    </svg>
  )
}

/**
 * Returns the CSS padding-top needed so content doesn't hide behind the Linux titlebar.
 * Returns 0 on macOS/Windows/browser.
 */
export function useLinuxTitlebarOffset(): number {
  return isLinux ? TITLEBAR_HEIGHT : 0
}
