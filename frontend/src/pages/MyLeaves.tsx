import React, { useEffect, useRef, useState } from 'react';
import { format } from 'date-fns';
import { Calendar, Filter, History, FileText, GitBranch } from 'lucide-react';
import { useSearchParams } from 'react-router-dom';
import api from '../services/api';
import type { LeaveRequest } from '../types';
import { Card } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { getDisplayDuration, formatDuration } from '../utils/duration';
import LeaveHistoryModal from '../components/LeaveHistoryModal';
import WorkflowStateDisplay from '../components/WorkflowStateDisplay';
import { ConfirmationModal } from '../components/ui/ConfirmationModal';
import { useToast } from '../components/ui/Toast';

const MyLeaves: React.FC = () => {
    const { showToast } = useToast();
    const [searchParams, setSearchParams] = useSearchParams();
    const [requests, setRequests] = useState<LeaveRequest[]>([]);
    const [loading, setLoading] = useState(true);
    const [filter, setFilter] = useState<'all' | 'pending' | 'approved' | 'rejected'>('all');
    const [historyModalOpen, setHistoryModalOpen] = useState(false);
    const [selectedRequestId, setSelectedRequestId] = useState<string | null>(null);
    const [selectedLeaveType, setSelectedLeaveType] = useState<string>('');
    const [showWorkflow, setShowWorkflow] = useState<Record<string, boolean>>({});
    const requestRowRefs = useRef<Record<string, HTMLTableRowElement | null>>({});

    // Confirmation Modal State
    const [confirmModalOpen, setConfirmModalOpen] = useState(false);
    const [pendingCancelId, setPendingCancelId] = useState<string | null>(null);
    const [processingId, setProcessingId] = useState<string | null>(null);

    const openHistoryModal = (req: LeaveRequest) => {
        setSelectedRequestId(req.id);
        setSelectedLeaveType(req.leave_type);
        setHistoryModalOpen(true);
    };

    const fetchRequests = async () => {
        setLoading(true);
        try {
            const response = await api.get('/leave-requests');
            if (Array.isArray(response.data)) {
                setRequests(response.data);
            }
        } catch (error) {
            console.error("Failed to fetch leave requests", error);
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

    const initiateCancel = (id: string) => {
        setPendingCancelId(id);
        setConfirmModalOpen(true);
    };

    const handleConfirmCancel = async () => {
        if (!pendingCancelId) return;

        setProcessingId(pendingCancelId);
        try {
            await api.put(`/leave-requests/${pendingCancelId}/cancel`);
            showToast('Leave request cancelled', 'success');
            fetchRequests();
        } catch (error) {
            console.error("Failed to cancel request", error);
            showToast('Failed to cancel request', 'error');
        } finally {
            setProcessingId(null);
            setConfirmModalOpen(false);
            setPendingCancelId(null);
        }
    };

    const getStatusVariant = (status: string) => {
        switch (status) {
            case 'approved': return 'success';
            case 'rejected': return 'danger';
            case 'pending': return 'warning';
            case 'cancelled': return 'secondary';
            default: return 'default';
        }
    };

    const filteredRequests = requests.filter(req => {
        if (filter === 'all') return true;
        return req.status === filter;
    });

    if (loading) {
        return (
            <div className="flex justify-center items-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-brand-600"></div>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">My Leave History</h1>
                    <p className="text-slate-500 mt-1">View and manage your leave requests</p>
                </div>
                <div className="flex gap-2">
                    <div className="relative">
                        <select
                            className="appearance-none pl-9 pr-8 py-2 border border-slate-200 rounded-lg bg-white text-sm focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500 transition-all"
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
                                <th className="px-4 py-4 font-semibold text-slate-600">Type</th>
                                <th className="px-4 py-4 font-semibold text-slate-600">Period</th>
                                <th className="px-4 py-4 font-semibold text-slate-600">Duration</th>
                                <th className="px-4 py-4 font-semibold text-slate-600">Reason</th>
                                <th className="px-4 py-4 font-semibold text-slate-600">Status</th>
                                <th className="px-4 py-4 font-semibold text-slate-600">Applied On</th>
                                <th className="px-4 py-4 font-semibold text-slate-600 text-right">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {filteredRequests.length === 0 ? (
                                <tr>
                                    <td colSpan={7} className="px-4 py-12 text-center text-slate-500">
                                        <div className="flex flex-col items-center justify-center">
                                            <div className="bg-slate-100 p-3 rounded-full mb-3">
                                                <Calendar className="h-6 w-6 text-slate-400" />
                                            </div>
                                            <p className="font-medium text-slate-900">No leave requests found</p>
                                            <p className="text-sm mt-1">Create a new request to get started</p>
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
                                                <span className="capitalize font-medium text-slate-900">{req.leave_type}</span>
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
                                                    {req.status === 'pending' && req.workflow_state?.action_taken === 'requested_docs' && (
                                                        <span className="text-[10px] uppercase tracking-wide font-semibold text-amber-700 bg-amber-50 px-1.5 py-0.5 rounded border border-amber-200">
                                                            ⚠ Doc Requested
                                                        </span>
                                                    )}
                                                    {req.status === 'rejected' && req.rejection_reason && (
                                                        <span className="text-xs text-red-600 italic max-w-[150px] truncate" title={req.rejection_reason}>
                                                            "{req.rejection_reason}"
                                                        </span>
                                                    )}
                                                </div>
                                            </td>
                                            <td className="px-4 py-4 text-slate-500 text-xs">
                                                {format(new Date(req.created_at), 'MMM d, yyyy')}
                                            </td>
                                            <td className="px-4 py-4 text-right">
                                                <div className="flex justify-end gap-2">
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        className="h-8 w-8 p-0 rounded-full bg-purple-50 text-purple-600 hover:bg-purple-100"
                                                        onClick={() => setShowWorkflow(prev => ({ ...prev, [req.id]: !prev[req.id] }))}
                                                        title="View Workflow"
                                                    >
                                                        <GitBranch className="h-4 w-4" />
                                                    </Button>
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        className="text-slate-600 hover:text-slate-700 hover:bg-slate-50"
                                                        onClick={() => openHistoryModal(req)}
                                                    >
                                                        <History className="h-4 w-4" />
                                                    </Button>
                                                    {req.status === 'pending' && (
                                                        <Button
                                                            variant="ghost"
                                                            size="sm"
                                                            className="text-red-600 hover:text-red-700 hover:bg-red-50"
                                                            onClick={() => initiateCancel(req.id)}
                                                        >
                                                            Cancel
                                                        </Button>
                                                    )}
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
                                                        showActions={true} // Employee can upload docs if requested
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

            <ConfirmationModal
                isOpen={confirmModalOpen}
                onClose={() => setConfirmModalOpen(false)}
                onConfirm={handleConfirmCancel}
                title="Cancel Leave Request"
                message="Are you sure you want to cancel this leave request? This action cannot be undone."
                confirmText="Cancel Request"
                type="danger"
                isLoading={!!processingId}
            />
        </div>
    );
};

export default MyLeaves;
