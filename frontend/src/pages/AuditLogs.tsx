import React, { useEffect, useState } from 'react';
import { format } from 'date-fns';
import { Search } from 'lucide-react';
import api from '../services/api';
import { Card } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';
import { Input } from '../components/ui/Input';

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

const AuditLogDetailsModal: React.FC<{
    log: AuditLog | null;
    isOpen: boolean;
    onClose: () => void;
}> = ({ log, isOpen, onClose }) => {
    if (!isOpen || !log) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
            <div className="bg-white rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] flex flex-col">
                <div className="p-6 border-b border-slate-200 flex justify-between items-center">
                    <div>
                        <h2 className="text-xl font-bold text-slate-900">Audit Log Details</h2>
                        <p className="text-sm text-slate-500 mt-1">
                            {format(new Date(log.created_at), 'PPP pp')}
                        </p>
                    </div>
                    <button onClick={onClose} className="text-slate-400 hover:text-slate-600">
                        <span className="sr-only">Close</span>
                        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>

                <div className="flex-1 overflow-y-auto p-6 space-y-6">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div className="space-y-1">
                            <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">Actor</span>
                            <div className="flex items-center gap-2">
                                <span className="font-medium text-slate-900">{log.actor_email}</span>
                                <Badge variant="secondary">{log.actor_role}</Badge>
                            </div>
                        </div>
                        <div className="space-y-1">
                            <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">Action</span>
                            <div className="flex items-center gap-2">
                                <Badge variant="outline" className="font-mono">{log.method}</Badge>
                                <span className="font-mono text-sm text-slate-600">{log.endpoint}</span>
                            </div>
                        </div>
                    </div>

                    {(log.before_state || log.after_state) ? (
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                            <div className="space-y-2">
                                <h3 className="text-sm font-semibold text-slate-900 flex items-center gap-2">
                                    <span className="w-2 h-2 rounded-full bg-red-400"></span>
                                    Before Change
                                </h3>
                                <div className="bg-slate-50 rounded-md p-4 border border-slate-200 overflow-x-auto">
                                    {log.before_state ? (
                                        <pre className="text-xs font-mono text-slate-700">
                                            {JSON.stringify(log.before_state, null, 2)}
                                        </pre>
                                    ) : (
                                        <span className="text-sm text-slate-400 italic">No previous state recorded</span>
                                    )}
                                </div>
                            </div>
                            <div className="space-y-2">
                                <h3 className="text-sm font-semibold text-slate-900 flex items-center gap-2">
                                    <span className="w-2 h-2 rounded-full bg-green-400"></span>
                                    After Change
                                </h3>
                                <div className="bg-slate-50 rounded-md p-4 border border-slate-200 overflow-x-auto">
                                    {log.after_state ? (
                                        <pre className="text-xs font-mono text-slate-700">
                                            {JSON.stringify(log.after_state, null, 2)}
                                        </pre>
                                    ) : (
                                        <span className="text-sm text-slate-400 italic">No new state recorded</span>
                                    )}
                                </div>
                            </div>
                        </div>
                    ) : (
                        <div className="bg-slate-50 rounded-lg p-8 text-center border border-dashed border-slate-300">
                            <p className="text-slate-500">No detailed state changes were captured for this event.</p>
                        </div>
                    )}
                </div>

                <div className="p-6 border-t border-slate-200 bg-slate-50 rounded-b-lg">
                    <button
                        onClick={onClose}
                        className="w-full sm:w-auto px-4 py-2 bg-white border border-slate-300 rounded-md text-sm font-medium text-slate-700 hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                    >
                        Close Details
                    </button>
                </div>
            </div>
        </div>
    );
};

const AuditLogs: React.FC = () => {
    const [logs, setLogs] = useState<AuditLog[]>([]);
    const [loading, setLoading] = useState(true);
    const [searchQuery, setSearchQuery] = useState('');
    const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null);

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

    const getMethodVariant = (method: string) => {
        switch (method) {
            case 'GET': return 'primary';
            case 'POST': return 'success';
            case 'PUT': return 'warning';
            case 'DELETE': return 'danger';
            default: return 'default';
        }
    };

    const filteredLogs = logs.filter(log =>
        (log.endpoint || log.action || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
        (log.actor_email || '').toLowerCase().includes(searchQuery.toLowerCase())
    );

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">System Audit Logs</h1>
                    <p className="text-slate-500 mt-1">Monitor system activity and security events</p>
                </div>
            </div>

            <Card className="p-4">
                <div className="flex items-center gap-2 mb-4">
                    <div className="relative flex-1 max-w-sm">
                        <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
                        <Input
                            placeholder="Search logs..."
                            className="pl-9"
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                        />
                    </div>
                </div>

                <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                        <thead>
                            <tr className="bg-slate-50 border-b border-slate-200">
                                <th className="px-6 py-4 font-semibold text-slate-600">Time</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">User</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Role</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">Action</th>
                                <th className="px-6 py-4 font-semibold text-slate-600">IP Address</th>
                                <th className="px-6 py-4 font-semibold text-slate-600 text-right">Details</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {loading ? (
                                <tr>
                                    <td colSpan={6} className="px-6 py-12 text-center text-slate-500">
                                        Loading logs...
                                    </td>
                                </tr>
                            ) : filteredLogs.length === 0 ? (
                                <tr>
                                    <td colSpan={6} className="px-6 py-12 text-center text-slate-500">
                                        No logs found. Try performing some actions first (navigate around, submit leave, etc.)
                                    </td>
                                </tr>
                            ) : (
                                filteredLogs.map((log) => (
                                    <tr key={log.id} className="hover:bg-slate-50 transition-colors text-xs">
                                        <td className="px-6 py-4 whitespace-nowrap text-slate-600 font-mono">
                                            {format(new Date(log.created_at), 'MMM d, HH:mm:ss')}
                                        </td>
                                        <td className="px-6 py-4 text-slate-900 font-medium">
                                            {log.actor_email || 'System'}
                                        </td>
                                        <td className="px-6 py-4">
                                            <Badge variant="secondary" className="capitalize">
                                                {log.actor_role || 'N/A'}
                                            </Badge>
                                        </td>
                                        <td className="px-6 py-4 text-slate-600">
                                            <div className="flex items-center gap-2">
                                                {log.method && (
                                                    <Badge variant={getMethodVariant(log.method) as any} className="text-[10px]">
                                                        {log.method}
                                                    </Badge>
                                                )}
                                                <span className="truncate max-w-xs font-mono" title={log.action}>
                                                    {log.endpoint || log.action}
                                                </span>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 text-slate-500 font-mono">
                                            {log.ip_address || '-'}
                                        </td>
                                        <td className="px-6 py-4 text-right">
                                            <button
                                                onClick={() => setSelectedLog(log)}
                                                className="text-indigo-600 hover:text-indigo-900 font-medium"
                                            >
                                                View
                                            </button>
                                        </td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </Card>

            <AuditLogDetailsModal
                log={selectedLog}
                isOpen={!!selectedLog}
                onClose={() => setSelectedLog(null)}
            />
        </div>
    );
};

export default AuditLogs;

