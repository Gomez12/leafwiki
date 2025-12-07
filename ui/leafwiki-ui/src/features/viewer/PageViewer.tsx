import Page404 from '@/components/Page404'
import {
  DIALOG_COPY_PAGE,
  DIALOG_DELETE_PAGE_CONFIRMATION,
  DIALOG_PAGE_HISTORY,
} from '@/lib/registries'
import { useScrollRestoration } from '@/lib/useScrollRestoration'
import { useDialogsStore } from '@/stores/dialogs'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import MarkdownPreview from '../preview/MarkdownPreview'
import { useProgressbarStore } from '../progressbar/progressbar'
import Breadcrumbs from './Breadcrumbs'
import { useScrollToHeadline } from './useScrollToHeadline'
import { useSetPageTitle } from './useSetPageTitle'
import { useToolbarActions } from './useToolbarActions'
import { useViewerStore } from './viewer'
import { getPageHistory, PageHistoryEntry } from '@/lib/api/pages'

export default function PageViewer() {
  const { pathname } = useLocation()
  const navigate = useNavigate()
  const openDialog = useDialogsStore((state) => state.openDialog)
  const loading = useProgressbarStore((s) => s.loading)
  const error = useViewerStore((s) => s.error)
  const page = useViewerStore((s) => s.page)
  const loadPageData = useViewerStore((s) => s.loadPageData)
  const [historyEntries, setHistoryEntries] = useState<PageHistoryEntry[]>([])
  const [historyHash, setHistoryHash] = useState('')
  const [historyLoading, setHistoryLoading] = useState(false)

  const actions = {
    printPage: useCallback(() => {
      window.print()
    }, []),
    editPage: useCallback(() => {
      navigate(`/e/${page?.path || ''}`)
    }, [page?.path, navigate]),
    deletePage: useCallback(() => {
      const redirectUrl = page?.path.split('/').slice(0, -1).join('/')
      openDialog(DIALOG_DELETE_PAGE_CONFIRMATION, {
        pageId: page?.id,
        redirectUrl,
      })
    }, [page, openDialog]),
    copyPage: useCallback(() => {
      if (!page) return
      openDialog(DIALOG_COPY_PAGE, { sourcePage: page })
    }, [page, openDialog]),
  }

  const openHistoryDialog = useCallback(() => {
    if (!page) return
    openDialog(DIALOG_PAGE_HISTORY, {
      history: historyEntries,
      currentHash: historyHash,
      pageTitle: page.title,
      pagePath: page.path,
    })
  }, [historyEntries, historyHash, openDialog, page])

  useScrollRestoration(pathname, loading)
  useScrollToHeadline({ content: page?.content || '', isLoading: loading })
  useSetPageTitle({ page })

  useEffect(() => {
    const path = pathname.slice(1) // remove leading /
    loadPageData?.(path)
  }, [pathname, loadPageData])

  useEffect(() => {
    if (!page?.path) {
      setHistoryEntries([])
      setHistoryHash('')
      return
    }

    let isCancelled = false
    setHistoryLoading(true)
    ;(async () => {
      try {
        const data = await getPageHistory(page.path)
        if (isCancelled) return
        setHistoryEntries(data.history || [])
        setHistoryHash(data.currentHash || '')
      } catch (err) {
        console.warn('Failed to load history', err)
        if (isCancelled) return
        setHistoryEntries([])
        setHistoryHash('')
      } finally {
        if (!isCancelled) {
          setHistoryLoading(false)
        }
      }
    })()

    return () => {
      isCancelled = true
    }
  }, [page?.path])

  const disableHistoryButton = useMemo(() => {
    if (historyLoading) return true
    return historyEntries.length <= 1
  }, [historyEntries.length, historyLoading])

  useToolbarActions({
    ...actions,
    viewHistory: {
      action: openHistoryDialog,
      disabled: disableHistoryButton,
    },
  })

  const renderError = () => {
    if (!loading && !page) {
      return <Page404 />
    }
    if (!loading && error) {
      return <p className="page-viewer__error">Error: {error}</p>
    }
    return null
  }

  return (
    <div className="page-viewer">
      <div>
        <Breadcrumbs />
      </div>

      {/* we keep the content also during loading to avoid flickering */}
      {page && !error && (
        <article className="page-viewer__content">
          <MarkdownPreview content={page.content} path={page.path} />
        </article>
      )}
      {renderError()}
    </div>
  )
}
