import React, { useEffect, useRef, useState } from 'react';
import { format } from 'date-fns';
import { Filter, Check, History, FileText, GitBranch } from 'lucide-react';
import { useSearchParams } from 'react-router-dom';
import api from '../services/api';
import type { LeaveRequest } from '../types';
import { Card } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { getDisplayDuration, formatDuration } from '../utils/duration';
import LeaveHistoryModal from '../components/LeaveHistoryModal';
import WorkflowStateDisplay from '../components/WorkflowStateDisplay';

const TeamLeaves: React.FC = () => {
    const [searchParams, setSearchParams] = useSearchParams();
    const [requests, setRequests] = useState<LeaveRequest[]>([]);
    const [loading, setLoading] = useState(true);
    const [filter, setFilter] = useState<'all' | 'pending' | 'approved' | 'rejected'>('pending');

    // Modal State
    const [historyModalOpen, setHistoryModalOpen] = useState(false);
    const [selectedRequestId, setSelectedRequestId] = useState<string | null>(null);
    const [selectedLeaveType, setSelectedLeaveType] = useState<string>('');
    const [showWorkflow, setShowWorkflow] = useState<Record<string, boolean>>({});
    const requestRowRefs = useRef<Record<string, HTMLTableRowElement | null>>({});

    const openHistoryModal = (req: LeaveRequest) => {
        setSelectedRequestId(req.id);
        setSelectedLeaveType(req.leave_type);
        setHistoryModalOpen(true);
    };

    const fetchRequests = async () => {
        setLoading(true);
        try {
            const response = await api.get('/team/leave-requests');
            if (Array.isArray(response.data)) {
                setRequests(response.data);
            }
        } catch (error) {
            console.error("Failed to fetch team requests", error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchRequests();
    }, []);

    useEffect(() => {
        const requestId = searchParams.get('requestId');
        if (!requestId || requests.length === 0) return;

        const matchingRequest = requests.find(req => req.id === requestId);
        if (!matchingRequest) return;

        setShowWorkflow(prev => ({ ...prev, [requestId]: true }));
        setSearchParams(prev => {
            const next = new URLSearchParams(prev);
            next.delete('requestId');
            return next;
        }, { replace: true });
    }, [requests, searchParams, setSearchParams]);

    useEffect(() => {
        const expandedRequestId = Object.keys(showWorkflow).find(id => showWorkflow[id]);
        if (!expandedRequestId) return;

        const row = requestRowRefs.current[expandedRequestId];
        if (!row) return;

        row.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }, [showWorkflow]);

    const getStatusVariant = (status: string) => {
        switch (status) {
            case 'approved': return 'success';
            case 'rejected': return 'danger';
            case 'pending': return 'warning';
            case 'escalated': return 'warning';
            case 'cancelled': return 'secondary';
            default: return 'default';
        }
    };

    if (loading) {
        return (
            <div className="flex justify-center items-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-600"></div>
            </div>
        );
    }

    const filteredRequests = requests.filter(req => {
        if (filter === 'all') return true;
        return req.status === filter;
    });

    return (
        <div className="space-y-6">
            <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">Team Requests</h1>
                    <p className="text-slate-500 mt-1">Manage leave requests from your team members</p>
                </div>
                <div className="flex gap-2 w-full sm:w-auto">
                    <div className="relative flex-1 sm:flex-none">
                        <select
                            className="w-full appearance-none pl-9 pr-8 py-2 border border-slate-200 rounded-lg bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500 transition-all"
                            value={filter}
                            onChange={(e) => setFilter(e.target.value as any)}
                        >
                            <option value="all">All Status</option>
                            <option value="pending">Pending</option>
                            <option value="approved">Approved</option>
                            <option value="rejected">Rejected</option>
                        </select>
                        <Filter className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
                    </div>
                </div>
            </div>

            <Card>
                <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                        <thead>
                            <tr className="bg-slate-50 border-b border-slate-200">
                                <th className="px-4 py-4 font-semibold text-slate-600">Employee</th>
                                <th className="px-4 py-4 font-semibold text-slate-600">Type</th>
                                <th className="px-4 py-4 font-semibold text-slate-600">Period</th>
                                <th className="px-4 py-4 font-semibold text-slate-600">Duration</th>
                                <th className="px-4 py-4 font-semibold text-slate-600">Reason</th>
                                <th className="px-4 py-4 font-semibold text-slate-600">Status</th>
                                <th className="px-4 py-4 font-semibold text-slate-600 text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {filteredRequests.length === 0 ? (
                                <tr>
                                    <td colSpan={7} className="px-4 py-12 text-center text-slate-500">
                                        <div className="flex flex-col items-center justify-center">
                                            <div className="bg-slate-100 p-3 rounded-full mb-3">
                                                <Check className="h-6 w-6 text-slate-400" />
                                            </div>
                                            <p className="font-medium text-slate-900">No requests found</p>
                                            <p className="text-sm mt-1">Great! You're all caught up.</p>
                                        </div>
                                    </td>
                                </tr>
                            ) : (
                                filteredRequests.map((req) => (
                                    <React.Fragment key={req.id}>
                                        <tr
                                            ref={(element) => { requestRowRefs.current[req.id] = element; }}
                                            className="hover:bg-slate-50 transition-colors"
                                        >
                                            <td className="px-4 py-4">
                                                <div className="flex items-center gap-3">
                                                    <div className="h-8 w-8 rounded-full bg-brand-100 text-brand-600 flex items-center justify-center font-bold text-xs shrink-0">
                                                        {req.user?.first_name?.[0]}{req.user?.last_name?.[0]}
                                                    </div>
                                                    <div className="min-w-0">
                                                        <p className="font-medium text-slate-900 truncate">{req.user?.first_name} {req.user?.last_name}</p>
                                                        <p className="text-xs text-slate-500 truncate max-w-[120px]">{req.user?.email}</p>
                                                    </div>
                                                </div>
                                            </td>
                                            <td className="px-4 py-4">
                                                <span className="capitalize font-medium text-slate-700">{req.leave_type}</span>
                                            </td>
                                            <td className="px-4 py-4 text-slate-600 whitespace-nowrap">
                                                <div className="flex flex-col">
                                                    <div className="flex items-center gap-2">
                                                        <span>{format(new Date(req.start_date), 'MMM d, yyyy')}</span>
                                                        {req.is_half_day && (
                                                            <span className="px-1.5 py-0.5 rounded text-[10px] uppercase font-semibold bg-brand-50 text-brand-700">
                                                                Half-Day {req.half_day_period}
                                                            </span>
                                                        )}
                                                    </div>
                                                    {!req.is_half_day && (
                                                        <span className="text-xs text-slate-400">to {format(new Date(req.end_date), 'MMM d, yyyy')}</span>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-4 py-4 text-slate-600">
                                                {formatDuration(getDisplayDuration(req.duration_days, req.start_date, req.end_date))}
                                            </td>
                                            <td className="px-4 py-4 text-slate-600 max-w-[200px]" title={req.reason}>
                                                <div className="flex items-center gap-2">
                                                    <span className="truncate">{req.reason}</span>
                                                    {req.attachment_url && (
                                                        <a
                                                            href={req.attachment_url}
                                                            target="_blank"
                                                            rel="noopener noreferrer"
                                                            className="text-brand-600 hover:text-brand-700 inline-flex items-center shrink-0"
                                                            title="View Attachment"
                                                        >
                                                            <FileText className="h-4 w-4" />
                                                        </a>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-4 py-4">
                                                <div className="flex flex-col gap-1">
                                                    <Badge variant={getStatusVariant(req.status)}>
                                                        {req.status}
                                                    </Badge>
                                                    {req.status === 'pending' && req.workflow_state?.current_step && (
                                                        <span className="text-[10px] uppercase tracking-wide font-semibold text-slate-500 bg-slate-100 px-1.5 py-0.5 rounded border border-slate-200">
                                                            Waiting: {req.workflow_state.current_step.responsible_role}
                                                        </span>
                                                    )}
                                                    {req.status === 'rejected' && req.rejection_reason && (
                                                        <span className="text-xs text-red-600 italic max-w-[150px] truncate" title={req.rejection_reason}>
                                                            "{req.rejection_reason}"
                                                        </span>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-4 py-4 text-right">
                                                <div className="flex justify-end gap-2">
                                                    <Button
                                                        size="sm"
                                                        className="h-8 w-8 p-0 rounded-full bg-purple-50 text-purple-600 hover:bg-purple-100 border-purple-200"
                                                        variant="ghost"
                                                        onClick={() => setShowWorkflow(prev => ({ ...prev, [req.id]: !prev[req.id] }))}
                                                        title="View Workflow"
                                                    >
                                                        <GitBranch className="h-4 w-4" />
                                                    </Button>
                                                    <Button
                                                        size="sm"
                                                        className="h-8 w-8 p-0 rounded-full bg-slate-50 text-slate-600 hover:bg-slate-100 border-slate-200"
                                                        variant="ghost"
                                                        onClick={() => openHistoryModal(req)}
                                                    >
                                                        <History className="h-4 w-4" />
                                                    </Button>
                                                </div>
                                            </td>
                                        </tr>
                                        {showWorkflow[req.id] && (
                                            <tr className="bg-slate-50">
                                                <td colSpan={7} className="px-4 py-3">
                                                    <WorkflowStateDisplay
                                                        requestId={req.id}
                                                        applicantId={req.user_id}
                                                        currentStatus={req.status}
                                                        onActionComplete={fetchRequests}
                                                        showActions={req.status === 'pending' && !!req.can_action}
                                                        isNegativeBalance={req.is_negative_balance}
                                                    />
                                                </td>
                                            </tr>
                                        )}
                                    </React.Fragment>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </Card>

            <LeaveHistoryModal
                isOpen={historyModalOpen}
                onClose={() => setHistoryModalOpen(false)}
                requestId={selectedRequestId}
                leaveType={selectedLeaveType}
            />
        </div>
    );
};

export default TeamLeaves;
