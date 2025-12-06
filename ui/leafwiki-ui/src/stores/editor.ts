import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type EditorStore = {
  previewVisible: boolean
  setPreviewVisible: (visible: boolean) => void
  togglePreview: () => void
  autosaveEnabled: boolean
  setAutosaveEnabled: (enabled: boolean) => void
}

export const useEditorStore = create<EditorStore>()(
  persist(
    (set, get) => ({
      previewVisible: true,
      setPreviewVisible: (visible) => set({ previewVisible: visible }),
      togglePreview: () => set({ previewVisible: !get().previewVisible }),
      autosaveEnabled: true,
      setAutosaveEnabled: (enabled) => set({ autosaveEnabled: enabled }),
    }),
    {
      name: 'leafwiki-editor-settings', // localStorage-Key
      partialize: (state) => ({
        previewVisible: state.previewVisible,
        autosaveEnabled: state.autosaveEnabled,
      }),
    },
  ),
)
