import Page404 from '@/components/Page404'
import { Page } from '@/lib/api/pages'
import { buildEditUrl } from '@/lib/urlUtil'
import { useEditorStore } from '@/stores/editor'
import { useTreeStore } from '@/stores/tree'
import { useCallback, useEffect, useRef } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { useProgressbarStore } from '../progressbar/progressbar'
import MarkdownEditor, { MarkdownEditorRef } from './MarkdownEditor'
import { usePageEditorStore } from './pageEditor'
import useNavigationGuard from './useNavigationGuard'
import { useToolbarActions } from './useToolbarActions'

export default function PageEditor() {
  const { '*': path } = useParams()
  const AUTOSAVE_DELAY_MS = 500

  const navigate = useNavigate()
  const editorRef = useRef<MarkdownEditorRef>(null)
  const reloadTree = useTreeStore((s) => s.reloadTree)
  const savePage = usePageEditorStore((s) => s.savePage)
  const setContent = usePageEditorStore((s) => s.setContent)
  const loadPageData = usePageEditorStore((s) => s.loadPageData)
  const initialPage = usePageEditorStore((s) => s.initialPage) // contains the initial page data when loaded
  const loading = useProgressbarStore((s) => s.loading)
  const error = usePageEditorStore((s) => s.error)
  const page = usePageEditorStore((s) => s.page)
  const dirty = usePageEditorStore((s) => {
    const { page, title, slug, content } = s
    if (!page) return false
    return (
      page.title !== title || page.slug !== slug || page.content !== content
    )
  })
  const title = usePageEditorStore((s) => s.title)
  const slug = usePageEditorStore((s) => s.slug)
  const content = usePageEditorStore((s) => s.content)
  const autosaveEnabled = useEditorStore((s) => s.autosaveEnabled)
  const autosaveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const ongoingSaveRef = useRef<Promise<Page | null | undefined> | null>(null)

  // Shows Unsaved Changes Dialog when navigating away with dirty state
  useNavigationGuard({
    when: dirty,
    onNavigate: async () => {
      await reloadTree()
    },
  })

  // Load page data when path changes
  useEffect(() => {
    if (!path) return
    loadPageData(path)
  }, [path, loadPageData])

  const saveAndUpdateUrl = useCallback(
    async ({ silent = false }: { silent?: boolean } = {}) => {
      if (ongoingSaveRef.current) {
        await ongoingSaveRef.current
      }

      const pendingSave = (async () => {
        try {
          const savedPage = await savePage()
          if (savedPage) {
            window.history.replaceState(
              null,
              '',
              buildEditUrl(`/${savedPage?.path}`),
            )
            if (!silent) {
              toast.success('Page saved successfully')
            }
          }
          return savedPage
        } catch (err) {
          toast.error(silent ? 'Autosave failed' : 'Error saving page')
          return null
        }
      })()

      ongoingSaveRef.current = pendingSave

      try {
        return await pendingSave
      } finally {
        ongoingSaveRef.current = null
      }
    },
    [savePage],
  )

  // callbacks to save / close
  const handleSave = useCallback(() => {
    // clear any pending autosave when saving manually
    if (autosaveTimerRef.current) {
      clearTimeout(autosaveTimerRef.current)
      autosaveTimerRef.current = null
    }
    void saveAndUpdateUrl()
  }, [saveAndUpdateUrl])

  const handleClose = useCallback(() => {
    if (page?.path) {
      navigate(`/${page.path}`)
    } else {
      navigate('/')
    }
  }, [page, navigate])

  // register toolbar actions
  useToolbarActions({
    savePage: () => handleSave(),
    closePage: handleClose,
  })

  // content changes in the editor are synced to the store
  const handleEditorChange = useCallback(
    (value: string) => {
      setContent(value) // store update
    },
    [setContent],
  )

  // Autosave after 2 seconds of inactivity when there are unsaved changes
  useEffect(() => {
    if (autosaveTimerRef.current) {
      clearTimeout(autosaveTimerRef.current)
      autosaveTimerRef.current = null
    }

    if (!autosaveEnabled || !dirty || !page) {
      return
    }

    autosaveTimerRef.current = setTimeout(() => {
      void saveAndUpdateUrl({ silent: true })
    }, AUTOSAVE_DELAY_MS)

    return () => {
      if (autosaveTimerRef.current) {
        clearTimeout(autosaveTimerRef.current)
        autosaveTimerRef.current = null
      }
    }
  }, [
    autosaveEnabled,
    dirty,
    title,
    slug,
    content,
    page,
    saveAndUpdateUrl,
  ])

  if (error) return <p className="page-editor__error">Error: {error}</p>

  if (!initialPage && !loading)
    return (
      <div className="page-editor__not-found">
        <Page404 />
      </div>
    )

  return (
    <>
      <div className="page-editor">
        {initialPage && (
          <MarkdownEditor
            ref={editorRef}
            pageId={initialPage.id}
            initialValue={initialPage.content || ''}
            onChange={handleEditorChange}
          />
        )}
      </div>
    </>
  )
}
