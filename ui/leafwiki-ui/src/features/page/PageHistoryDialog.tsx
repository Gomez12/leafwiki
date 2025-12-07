import BaseDialog from '@/components/BaseDialog'
import { PageHistoryEntry } from '@/lib/api/pages'
import { DIALOG_PAGE_HISTORY } from '@/lib/registries'
import { CalendarClock, GitCommitVertical, FileText } from 'lucide-react'

type PageHistoryDialogProps = {
  history: PageHistoryEntry[]
  currentHash: string
  pageTitle: string
  pagePath: string
}

export function PageHistoryDialog({
  history,
  currentHash,
  pageTitle,
  pagePath,
}: PageHistoryDialogProps) {
  const title = pageTitle || 'Untitled'
  const formatDate = (value: string) => {
    if (!value) return 'Unknown time'
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value
    return date.toLocaleString()
  }

  const statusLabel = (status: PageHistoryEntry['status']) => {
    switch (status) {
      case 'created':
        return 'Created'
      case 'modified':
        return 'Modified'
      case 'moved':
        return 'Moved'
      case 'deleted':
        return 'Deleted'
      default:
        return status
    }
  }

  const statusTone = (status: PageHistoryEntry['status']) => {
    switch (status) {
      case 'created':
        return 'text-emerald-700 bg-emerald-50'
      case 'modified':
        return 'text-blue-700 bg-blue-50'
      case 'moved':
        return 'text-amber-700 bg-amber-50'
      case 'deleted':
        return 'text-red-700 bg-red-50'
      default:
        return 'text-slate-700 bg-slate-100'
    }
  }

  return (
    <BaseDialog
      dialogType={DIALOG_PAGE_HISTORY}
      dialogTitle={`History for "${title}"`}
      dialogDescription={`Path: /${pagePath}`}
      contentClassName="max-w-5xl w-full"
      defaultAction="cancel"
      onClose={() => true}
      onConfirm={async () => true}
      cancelButton={{ label: 'Close', variant: 'outline', autoFocus: true }}
      buttons={[]}
    >
      <div className="space-y-4 pt-2 max-h-[70vh] overflow-auto pr-1">
        {history.length === 0 && (
          <p className="text-sm text-slate-600">
            No history recorded for this page yet.
          </p>
        )}
        {history.map((entry) => {
          const isCurrent = entry.hash === currentHash && entry.status !== 'deleted'
          return (
            <div
              key={entry.id}
              className="flex items-start gap-3 rounded-md border border-slate-200 bg-white p-3 shadow-sm dark:border-slate-700 dark:bg-slate-900"
            >
              <div className="mt-0.5 text-slate-400 dark:text-slate-500">
                <GitCommitVertical size={18} />
              </div>
              <div className="flex-1 space-y-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs font-semibold ${statusTone(entry.status)}`}
                  >
                    {statusLabel(entry.status)}
                  </span>
                  <span className="inline-flex items-center gap-1 text-xs text-slate-600 dark:text-slate-400">
                    <CalendarClock size={14} />
                    {formatDate(entry.recordedAt)}
                  </span>
                  {isCurrent && (
                    <span className="text-xs text-emerald-600 dark:text-emerald-400">
                      Current
                    </span>
                  )}
                </div>
                <div className="text-sm text-slate-800 dark:text-slate-200">
                  <span className="font-semibold">Path:</span>{' '}
                  <code className="rounded bg-slate-100 px-1 py-0.5 dark:bg-slate-800">
                    {entry.path}
                  </code>
                  {entry.previousPath && (
                    <span className="ml-2 text-slate-500 dark:text-slate-400">
                      (from <code>{entry.previousPath}</code>)
                    </span>
                  )}
                </div>
                <div className="text-xs text-slate-600 dark:text-slate-400">
                  Hash:{' '}
                  <code className="rounded bg-slate-100 px-1 py-0.5 dark:bg-slate-800">
                    {entry.hash.slice(0, 12)}
                  </code>
                </div>
                <div className="flex items-start gap-2 text-sm text-slate-700 dark:text-slate-200">
                  <FileText size={14} className="mt-0.5 text-slate-400 dark:text-slate-500" />
                  <pre className="max-h-64 w-full overflow-auto whitespace-pre-wrap rounded border border-slate-200 bg-slate-50 p-3 text-xs leading-relaxed dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100">
                    {entry.content || '—'}
                  </pre>
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </BaseDialog>
  )
}
