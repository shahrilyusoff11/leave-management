import React, { useEffect, useState } from 'react';
import { format } from 'date-fns';
import { Search, Filter, Eye, X, Clock, User, Globe, Activity, ChevronLeft, ChevronRight } from 'lucide-react';
import api from '../services/api';
import { Input } from '../components/ui/Input';
import { getRoleLabel } from '../utils/roles';
import type { UserRole } from '../types';

interface AuditLog {
    id: string;
    method: string;
    endpoint: string;
    action: string;
    actor_email: string;
    actor_role: string;
    created_at: string;
    ip_address: string;
    target_type?: string;
    target_id?: string;
    before_state?: Record<string, any>;
    after_state?: Record<string, any>;
}

const METHOD_COLORS: Record<string, string> = {
    GET: 'bg-blue-100 text-blue-700',
    POST: 'bg-green-100 text-green-700',
    PUT: 'bg-amber-100 text-amber-700',
    DELETE: 'bg-red-100 text-red-700',
    PATCH: 'bg-purple-100 text-purple-700',
};

const ROWS_PER_PAGE = 15;

const AuditLogs: React.FC = () => {
    const [logs, setLogs] = useState<AuditLog[]>([]);
    const [loading, setLoading] = useState(true);
    const [searchQuery, setSearchQuery] = useState('');
    const [methodFilter, setMethodFilter] = useState<string>('all');
    const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);
    const [currentPage, setCurrentPage] = useState(1);

    const fetchLogs = async () => {
        setLoading(true);
        try {
            const response = await api.get('/admin/audit-logs');
            if (response.data && response.data.logs) {
                setLogs(response.data.logs);
            } else if (Array.isArray(response.data)) {
                setLogs(response.data);
            } else {
                setLogs([]);
            }
        } catch (error) {
            console.error("Failed to fetch audit logs", error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchLogs();
    }, []);

    const filteredLogs = logs.filter(log => {
        const matchesSearch =
            (log.endpoint || log.action || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
            (log.actor_email || '').toLowerCase().includes(searchQuery.toLowerCase());
        const matchesMethod = methodFilter === 'all' || log.method === methodFilter;
        return matchesSearch && matchesMethod;
    });

    // Pagination
    const totalPages = Math.ceil(filteredLogs.length / ROWS_PER_PAGE);
    const startIdx = (currentPage - 1) * ROWS_PER_PAGE;
    const paginatedLogs = filteredLogs.slice(startIdx, startIdx + ROWS_PER_PAGE);

    // Reset to page 1 when filters change
    useEffect(() => {
        setCurrentPage(1);
    }, [searchQuery, methodFilter]);

    return (
        <div className="space-y-4">
            {/* Header */}
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">System Audit Logs</h1>
                    <p className="text-slate-500 text-sm mt-0.5">Monitor system activity and security events</p>
                </div>
            </div>

            {/* Filters */}
            <div className="flex items-center gap-3">
                <div className="relative flex-1 max-w-sm">
                    <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
                    <Input
                        placeholder="Search by endpoint or email..."
                        className="pl-9"
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                    />
                </div>
                <div className="flex items-center gap-1.5">
                    <Filter className="h-4 w-4 text-slate-400" />
                    <select
                        value={methodFilter}
                        onChange={(e) => setMethodFilter(e.target.value)}
                        className="h-10 px-3 rounded-lg border border-slate-200 bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500"
                    >
                        <option value="all">All Methods</option>
                        <option value="GET">GET</option>
                        <option value="POST">POST</option>
                        <option value="PUT">PUT</option>
                        <option value="DELETE">DELETE</option>
                    </select>
                </div>
                <div className="text-sm text-slate-500">
                    {filteredLogs.length} event{filteredLogs.length !== 1 ? 's' : ''}
                </div>
            </div>

            {/* Table */}
            <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
                <table className="w-full text-left text-xs">
                    <thead>
                        <tr className="bg-slate-50 border-b border-slate-200">
                            <th className="px-4 py-3 font-semibold text-slate-500 uppercase tracking-wider">Time</th>
                            <th className="px-4 py-3 font-semibold text-slate-500 uppercase tracking-wider">User</th>
                            <th className="px-4 py-3 font-semibold text-slate-500 uppercase tracking-wider">Role</th>
                            <th className="px-4 py-3 font-semibold text-slate-500 uppercase tracking-wider">Method</th>
                            <th className="px-4 py-3 font-semibold text-slate-500 uppercase tracking-wider">Endpoint</th>
                            <th className="px-4 py-3 font-semibold text-slate-500 uppercase tracking-wider">IP</th>
                            <th className="px-4 py-3 font-semibold text-slate-500 uppercase tracking-wider text-center">Info</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-50">
                        {loading ? (
                            <tr>
                                <td colSpan={7} className="px-4 py-16 text-center text-slate-400">
                                    <Activity className="h-6 w-6 mx-auto mb-2 animate-pulse" />
                                    Loading audit logs...
                                </td>
                            </tr>
                        ) : paginatedLogs.length === 0 ? (
                            <tr>
                                <td colSpan={7} className="px-4 py-16 text-center text-slate-400">
                                    No logs found
                                </td>
                            </tr>
                        ) : (
                            paginatedLogs.map((log) => (
                                <tr
                                    key={log.id}
                                    className="hover:bg-slate-50/70 transition-colors cursor-pointer"
                                    onClick={() => setSelectedLog(log)}
                                >
                                    <td className="px-4 py-2.5 whitespace-nowrap text-slate-500 font-mono">
                                        {format(new Date(log.created_at), 'MMM d, HH:mm:ss')}
                                    </td>
                                    <td className="px-4 py-2.5 text-slate-800 font-medium truncate max-w-[200px]" title={log.actor_email}>
                                        {log.actor_email || 'System'}
                                    </td>
                                    <td className="px-4 py-2.5">
                                        <span className="text-slate-500 capitalize">
                                            {log.actor_role ? getRoleLabel(log.actor_role as UserRole) : '—'}
                                        </span>
                                    </td>
                                    <td className="px-4 py-2.5">
                                        <span className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-bold tracking-wider ${METHOD_COLORS[log.method] || 'bg-slate-100 text-slate-600'}`}>
                                            {log.method}
                                        </span>
                                    </td>
                                    <td className="px-4 py-2.5 text-slate-600 font-mono truncate max-w-[300px]" title={log.endpoint || log.action}>
                                        {log.endpoint || log.action}
                                    </td>
                                    <td className="px-4 py-2.5 text-slate-400 font-mono">
                                        {log.ip_address || '—'}
                                    </td>
                                    <td className="px-4 py-2.5 text-center">
                                        <button
                                            onClick={(e) => { e.stopPropagation(); setSelectedLog(log); }}
                                            className="text-slate-400 hover:text-brand-600 transition-colors"
                                            title="View details"
                                        >
                                            <Eye className="h-3.5 w-3.5" />
                                        </button>
                                    </td>
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>

                {/* Pagination */}
                {totalPages > 1 && (
                    <div className="flex items-center justify-between px-4 py-3 border-t border-slate-200 bg-slate-50">
                        <p className="text-xs text-slate-500">
                            Showing {startIdx + 1}–{Math.min(startIdx + ROWS_PER_PAGE, filteredLogs.length)} of {filteredLogs.length}
                        </p>
                        <div className="flex items-center gap-1">
                            <button
                                onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                                disabled={currentPage === 1}
                                className="p-1.5 rounded-md hover:bg-slate-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                            >
                                <ChevronLeft className="h-4 w-4 text-slate-600" />
                            </button>
                            {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => {
                                let pageNum: number;
                                if (totalPages <= 5) {
                                    pageNum = i + 1;
                                } else if (currentPage <= 3) {
                                    pageNum = i + 1;
                                } else if (currentPage >= totalPages - 2) {
                                    pageNum = totalPages - 4 + i;
                                } else {
                                    pageNum = currentPage - 2 + i;
                                }
                                return (
                                    <button
                                        key={pageNum}
                                        onClick={() => setCurrentPage(pageNum)}
                                        className={`min-w-[28px] h-7 rounded-md text-xs font-medium transition-colors ${currentPage === pageNum
                                                ? 'bg-brand-600 text-white'
                                                : 'text-slate-600 hover:bg-slate-200'
                                            }`}
                                    >
                                        {pageNum}
                                    </button>
                                );
                            })}
                            <button
                                onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                                disabled={currentPage === totalPages}
                                className="p-1.5 rounded-md hover:bg-slate-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
                            >
                                <ChevronRight className="h-4 w-4 text-slate-600" />
                            </button>
                        </div>
                    </div>
                )}
            </div>

            {/* Details slide-over panel */}
            {selectedLog && (
                <div className="fixed inset-0 z-50 flex justify-end" onClick={() => setSelectedLog(null)}>
                    <div className="absolute inset-0 bg-black/30 backdrop-blur-[2px]" />
                    <div
                        className="relative w-full max-w-lg bg-white shadow-2xl flex flex-col"
                        onClick={(e) => e.stopPropagation()}
                    >
                        {/* Panel header */}
                        <div className="flex-none flex items-center justify-between p-5 border-b border-slate-200">
                            <div>
                                <h2 className="text-lg font-bold text-slate-900">Log Details</h2>
                                <p className="text-xs text-slate-400 mt-0.5 font-mono">{selectedLog.id}</p>
                            </div>
                            <button
                                onClick={() => setSelectedLog(null)}
                                className="p-1.5 rounded-lg hover:bg-slate-100 text-slate-400 hover:text-slate-600 transition-colors"
                            >
                                <X className="h-5 w-5" />
                            </button>
                        </div>

                        {/* Panel body */}
                        <div className="flex-1 overflow-y-auto p-5 space-y-5">
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-1">
                                    <div className="flex items-center gap-1.5 text-xs text-slate-400 font-medium uppercase tracking-wider">
                                        <Clock className="h-3 w-3" /> Time
                                    </div>
                                    <p className="text-sm text-slate-800 font-mono">
                                        {format(new Date(selectedLog.created_at), 'PPP pp')}
                                    </p>
                                </div>
                                <div className="space-y-1">
                                    <div className="flex items-center gap-1.5 text-xs text-slate-400 font-medium uppercase tracking-wider">
                                        <Globe className="h-3 w-3" /> IP Address
                                    </div>
                                    <p className="text-sm text-slate-800 font-mono">{selectedLog.ip_address || '—'}</p>
                                </div>
                                <div className="space-y-1">
                                    <div className="flex items-center gap-1.5 text-xs text-slate-400 font-medium uppercase tracking-wider">
                                        <User className="h-3 w-3" /> Actor
                                    </div>
                                    <p className="text-sm text-slate-800">{selectedLog.actor_email || 'System'}</p>
                                    <span className="text-xs text-slate-500 capitalize">{selectedLog.actor_role ? getRoleLabel(selectedLog.actor_role as UserRole) : '—'}</span>
                                </div>
                                <div className="space-y-1">
                                    <div className="flex items-center gap-1.5 text-xs text-slate-400 font-medium uppercase tracking-wider">
                                        <Activity className="h-3 w-3" /> Action
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-bold ${METHOD_COLORS[selectedLog.method] || 'bg-slate-100 text-slate-600'}`}>
                                            {selectedLog.method}
                                        </span>
                                    </div>
                                    <p className="text-xs text-slate-600 font-mono break-all">{selectedLog.endpoint || selectedLog.action}</p>
                                </div>
                            </div>

                            {/* State changes */}
                            {(selectedLog.before_state || selectedLog.after_state) ? (
                                <div className="space-y-3">
                                    <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wider">State Changes</h3>

                                    {selectedLog.before_state && (
                                        <div className="space-y-1.5">
                                            <div className="flex items-center gap-2 text-xs font-medium text-red-600">
                                                <span className="w-2 h-2 rounded-full bg-red-400" />
                                                Before
                                            </div>
                                            <pre className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-[11px] font-mono text-slate-700 overflow-x-auto max-h-[200px] overflow-y-auto">
                                                {JSON.stringify(selectedLog.before_state, null, 2)}
                                            </pre>
                                        </div>
                                    )}

                                    {selectedLog.after_state && (
                                        <div className="space-y-1.5">
                                            <div className="flex items-center gap-2 text-xs font-medium text-green-600">
                                                <span className="w-2 h-2 rounded-full bg-green-400" />
                                                After
                                            </div>
                                            <pre className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-[11px] font-mono text-slate-700 overflow-x-auto max-h-[200px] overflow-y-auto">
                                                {JSON.stringify(selectedLog.after_state, null, 2)}
                                            </pre>
                                        </div>
                                    )}
                                </div>
                            ) : (
                                <div className="bg-slate-50 rounded-lg p-6 text-center border border-dashed border-slate-200">
                                    <p className="text-sm text-slate-400">No state changes captured for this event</p>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

export default AuditLogs;
