// Hook to provide toolbar actions for the page viewer

import { useAppMode } from '@/lib/useAppMode'
import { useIsReadOnly } from '@/lib/useIsReadOnly'
import { HotKeyDefinition, useHotKeysStore } from '@/stores/hotkeys'
import { Copy, History, Pencil, Printer, Trash2 } from 'lucide-react'
import { useEffect } from 'react'
import { useToolbarStore } from '../toolbar/toolbar'

export interface ToolbarActionsOptions {
  printPage: () => void
  editPage: () => void
  deletePage: () => void
  copyPage: () => void
  viewHistory?: {
    action: () => void
    disabled?: boolean
  }
}

// Hook to set up toolbar actions based on app mode and read-only status
export function useToolbarActions({
  printPage,
  editPage,
  deletePage,
  copyPage,
  viewHistory,
}: ToolbarActionsOptions) {
  const setButtons = useToolbarStore((state) => state.setButtons)
  const appMode = useAppMode()
  const readOnlyMode = useIsReadOnly()
  const registerHotkey = useHotKeysStore((s) => s.registerHotkey)
  const unregisterHotkey = useHotKeysStore((s) => s.unregisterHotkey)

  useEffect(() => {
    if (readOnlyMode || appMode !== 'view') {
      setButtons([])
      return
    }

    const buttons = [
      ...(viewHistory
        ? [
            {
              id: 'view-history',
              label: 'History',
              hotkey: 'Ctrl+H',
              icon: <History size={18} />,
              variant: 'outline',
              disabled: viewHistory.disabled,
              action: viewHistory.action,
            },
          ]
        : []),
      {
        id: 'delete-page',
        label: 'Delete Page',
        hotkey: 'Ctrl+Delete',
        icon: <Trash2 size={18} />,
        variant: 'outline',
        className: 'hover:text-red-600 hover:bg-red-100 hover:border-red-300',
        action: deletePage,
      },
      {
        id: 'copy-page',
        label: 'Copy Page',
        hotkey: 'Ctrl+Shift+S',
        icon: <Copy size={18} />,
        variant: 'outline',
        action: copyPage,
      },
      {
        id: 'print-page',
        label: 'Print Page',
        hotkey: 'Ctrl+P',
        icon: <Printer size={18} />,
        action: printPage,
      },
      {
        id: 'edit-page',
        label: 'Edit Page',
        hotkey: 'Ctrl+E',
        icon: <Pencil size={18} />,
        action: editPage,
      },
    ]

    setButtons(buttons)

    // Register hotkeys
    const historyHotkey: HotKeyDefinition | null = viewHistory
      ? {
          keyCombo: 'Mod+h',
          enabled: !viewHistory.disabled,
          mode: ['view'],
          action: viewHistory.action,
        }
      : null

    const copyHotkey: HotKeyDefinition = {
      keyCombo: 'Mod+Shift+S',
      enabled: true,
      mode: ['view'],
      action: copyPage,
    }

    const editHotkey: HotKeyDefinition = {
      keyCombo: 'Mod+e',
      enabled: true,
      mode: ['view'],
      action: editPage,
    }

    const deleteHotkey: HotKeyDefinition = {
      keyCombo: 'Mod+Delete',
      enabled: true,
      mode: ['view'],
      action: deletePage,
    }

    if (historyHotkey) registerHotkey(historyHotkey)
    registerHotkey(editHotkey)
    registerHotkey(copyHotkey)
    registerHotkey(deleteHotkey)

    return () => {
      if (historyHotkey) unregisterHotkey(historyHotkey.keyCombo)
      unregisterHotkey(editHotkey.keyCombo)
      unregisterHotkey(copyHotkey.keyCombo)
      unregisterHotkey(deleteHotkey.keyCombo)
    }
  }, [
    appMode,
    readOnlyMode,
    setButtons,
    deletePage,
    copyPage,
    editPage,
    printPage,
    registerHotkey,
    unregisterHotkey,
    viewHistory?.action,
    viewHistory?.disabled,
  ])
}
