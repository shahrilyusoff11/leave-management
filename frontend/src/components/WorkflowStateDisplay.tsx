import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useAuth } from '../context/AuthContext';
import { ChevronDown, CheckCircle, XCircle, FileQuestion, ArrowUpRight, History, AlertTriangle, RefreshCw } from 'lucide-react';
import { cn } from '../utils/cn';
import api from '../services/api';
import type { LeaveRequestWorkflowState } from '../types';
import { Badge } from './ui/Badge';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';
import { useToast } from './ui/Toast';
import './WorkflowState.css';

interface WorkflowStateDisplayProps {
    requestId: string;
    applicantId?: string;
    currentStatus: string;
    onActionComplete?: () => void;
    showActions?: boolean;
    isNegativeBalance?: boolean;
}

const actionLabels: Record<string, string> = {
    'pending': 'Pending',
    'approved': 'Approved',
    'rejected': 'Rejected',
    'verified': 'Verified',
    'not_verified': 'Not Verified',
    'requested_docs': 'Document Requested',
    'escalated': 'Escalated',
    'categorized_al': 'Categorized as AL',
    'categorized_unpaid': 'Categorized as Unpaid',
    'converted_to_el': 'Converted to EL',
    'converted_to_unpaid': 'Converted to Unpaid',
    'timeout_applied': 'Timeout Applied',
};

const WorkflowStateDisplay: React.FC<WorkflowStateDisplayProps> = ({
    requestId,
    applicantId,
    currentStatus,
    onActionComplete,
    showActions = true,
    isNegativeBalance = false
}) => {
    const { user } = useAuth();
    const isHR = user?.role === 'hr' || user?.role === 'admin' || user?.role === 'sysadmin';

    const { showToast } = useToast();
    const [workflowState, setWorkflowState] = useState<LeaveRequestWorkflowState | null>(null);
    const [loading, setLoading] = useState(true);
    const [actionLoading, setActionLoading] = useState(false);
    const [actionModalOpen, setActionModalOpen] = useState(false);
    const [selectedAction, setSelectedAction] = useState<string>('');
    const [actionComment, setActionComment] = useState('');
    const [conversionType, setConversionType] = useState('unpaid');
    const [uploading, setUploading] = useState(false);
    const fileInputRef = useRef<HTMLInputElement>(null);

    const fetchWorkflowState = useCallback(async () => {
        try {
            const response = await api.get<LeaveRequestWorkflowState>(`/leave-requests/${requestId}/workflow-state`);
            setWorkflowState(response.data);
        } catch (err) {
            // Workflow may not exist for this request
            setWorkflowState(null);
        } finally {
            setLoading(false);
        }
    }, [requestId]);

    useEffect(() => {
        fetchWorkflowState();
    }, [fetchWorkflowState]);

    const handleAction = async () => {
        if (!selectedAction) return;

        if (selectedAction === 'convert_leave_type' && !actionComment) {
            showToast('Reason is required for converting leave type', 'error');
            return;
        }

        setActionLoading(true);
        try {
            if (selectedAction === 'convert_leave_type') {
                await api.post(`/leave-requests/${requestId}/convert`, {
                    new_type: conversionType,
                    reason: actionComment
                });
            } else {
                await api.post(`/leave-requests/${requestId}/workflow-action`, {
                    action: selectedAction,
                    comment: actionComment
                });
            }
            showToast(`Action completed successfully`, 'success');
            setActionModalOpen(false);
            setActionComment('');
            fetchWorkflowState();
            onActionComplete?.();
        } catch (err: any) {
            showToast(err.response?.data?.error || 'Failed to perform action', 'error');
        } finally {
            setActionLoading(false);
        }
    };

    const handleDocUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        setUploading(true);
        try {
            // Step 1: Upload file to server
            const formData = new FormData();
            formData.append('file', file);
            const uploadRes = await api.post('/upload', formData, {
                headers: { 'Content-Type': 'multipart/form-data' }
            });

            // Step 2: Resubmit attachment on the leave request
            await api.put(`/leave-requests/${requestId}/resubmit-attachment`, {
                attachment_url: uploadRes.data.url,
                attachment_file_name: file.name
            });

            showToast('Document uploaded successfully', 'success');
            fetchWorkflowState();
            onActionComplete?.();
        } catch (err: any) {
            showToast(err.response?.data?.error || 'Failed to upload document', 'error');
        } finally {
            setUploading(false);
            if (fileInputRef.current) fileInputRef.current.value = '';
        }
    };

    const getAvailableActions = (): string[] => {
        if (!workflowState?.current_step) return [];

        const actionType = workflowState.current_step.action_type;
        switch (actionType) {
            case 'approve':
                return ['approve', 'reject', 'request_docs'];
            case 'verify':
                return ['verify', 'not_verify', 'request_docs'];
            case 'review':
                return ['approve', 'reject', 'escalate', 'request_docs'];
            case 'categorize':
                return ['categorize_al', 'categorize_unpaid'];
            default:
                return ['approve', 'reject'];
        }
    };

    const openActionModal = (action: string) => {
        setSelectedAction(action);
        setActionComment('');
        setActionModalOpen(true);
    };

    if (loading) {
        return <div className="workflow-state-loading">Loading...</div>;
    }

    if (!workflowState) {
        return null; // No workflow for this request
    }

    const isComplete = workflowState.is_complete;
    const currentStep = workflowState.current_step;

    return (
        <div className="workflow-state-container">
            <div className="workflow-state-header">
                <span className="workflow-label">Workflow Status</span>
                {isComplete ? (
                    <Badge variant={workflowState.final_status === 'approved' ? 'success' : 'danger'}>
                        {workflowState.final_status}
                    </Badge>
                ) : (
                    <Badge variant="warning">In Progress</Badge>
                )}
            </div>

            {isNegativeBalance && (
                <div className="mb-4 mt-2 p-3 bg-red-50 border border-red-200 rounded-lg flex items-start gap-3">
                    <AlertTriangle className="w-5 h-5 text-red-600 mt-0.5 shrink-0" />
                    <div>
                        <h4 className="text-sm font-bold text-red-800">Negative Balance Application</h4>
                        <p className="text-xs text-red-700 mt-1 leading-snug">
                            This leave request exceeds the employee's available leave balance. It has been flagged for mandatory HR review regardless of standard approval workflow.
                        </p>
                    </div>
                </div>
            )}

            {currentStep && !isComplete && (
                <div className="workflow-current-step">
                    <div className="step-info">
                        <span className="step-label">{currentStep.step_label || currentStep.step_name}</span>
                        <div className="flex items-center gap-2">
                            <span className="relative flex h-2 w-2">
                                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
                                <span className="relative inline-flex rounded-full h-2 w-2 bg-amber-500"></span>
                            </span>
                            <span className="step-role text-amber-600 font-medium">Awaiting: {currentStep.responsible_role}</span>
                        </div>
                    </div>

                    {/* Document Request Alert for Applicant Only */}
                    {workflowState.action_taken === 'requested_docs' && user?.id === applicantId && (
                        <div className="mt-3 p-3 bg-amber-50 border border-amber-200 rounded-lg">
                            <div className="flex items-start gap-2">
                                <span className="text-amber-600 font-bold text-sm">⚠️ Document Requested</span>
                            </div>
                            <p className="text-xs text-amber-700 mt-1">
                                Your approver has requested additional documents for this leave request.
                                Please upload the required document below.
                            </p>
                            {workflowState.action_comment && (
                                <p className="text-xs text-amber-800 mt-2 font-medium italic">
                                    "{workflowState.action_comment}"
                                </p>
                            )}
                            <div className="mt-3">
                                <input
                                    ref={fileInputRef}
                                    type="file"
                                    accept=".jpg,.jpeg,.png,.pdf,.doc,.docx"
                                    onChange={handleDocUpload}
                                    className="hidden"
                                    id={`doc-upload-${requestId}`}
                                />
                                <Button
                                    size="sm"
                                    variant="primary"
                                    isLoading={uploading}
                                    onClick={() => fileInputRef.current?.click()}
                                    className="bg-amber-600 hover:bg-amber-700 text-white"
                                >
                                    {uploading ? 'Uploading...' : '📎 Upload Document'}
                                </Button>
                            </div>
                        </div>
                    )}

                    {showActions && currentStatus === 'pending' && workflowState.action_taken !== 'requested_docs' && (
                        user?.role === currentStep.responsible_role || user?.role === 'sysadmin'
                    ) && (
                            <div className="workflow-actions mt-3">
                                {getAvailableActions().map(action => {
                                    const isReject = action.includes('reject') || action.includes('not');
                                    const isRequestDocs = action === 'request_docs';
                                    const isEscalate = action === 'escalate';

                                    let Icon = CheckCircle;
                                    let btnVariant = 'primary' as 'primary' | 'danger' | 'warning' | 'outline' | 'ghost';
                                    let btnClass = 'workflow-action-btn shadow-sm';

                                    if (isReject) {
                                        Icon = XCircle;
                                        btnVariant = 'danger';
                                    } else if (isRequestDocs) {
                                        Icon = FileQuestion;
                                        btnClass += ' bg-amber-500 hover:bg-amber-600 text-white border-none';
                                        btnVariant = 'primary'; // Base variant, overridden by classes
                                    } else if (isEscalate) {
                                        Icon = ArrowUpRight;
                                        btnClass += ' bg-purple-600 hover:bg-purple-700 text-white border-none';
                                        btnVariant = 'primary';
                                    }

                                    return (
                                        <Button
                                            key={action}
                                            size="sm"
                                            variant={btnVariant as any}
                                            onClick={() => openActionModal(action)}
                                            className={btnClass}
                                        >
                                            <Icon className="w-4 h-4 mr-1.5" />
                                            {action.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
                                        </Button>
                                    );
                                })}
                            </div>
                        )}
                    {isHR && !isComplete && (
                        <div className="mt-4 pt-4 border-t border-slate-100 flex justify-end">
                            <Button
                                size="sm"
                                variant="outline"
                                onClick={() => openActionModal('convert_leave_type')}
                                className="text-slate-600 border-slate-300 hover:bg-slate-50 shadow-sm"
                            >
                                <RefreshCw className="w-3.5 h-3.5 mr-1.5" />
                                Convert Leave Type
                            </Button>
                        </div>
                    )}
                </div>
            )}

            {workflowState.action_taken && workflowState.action_taken !== 'pending' && (
                <div className="workflow-last-action bg-slate-50 border border-slate-100 rounded-lg p-3 mt-4">
                    <div className="flex items-center gap-2 mb-1">
                        <History className="w-4 h-4 text-slate-400" />
                        <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">Last Action</span>
                    </div>
                    <div className="flex items-center gap-2 ml-6">
                        <div className="h-2 w-2 rounded-full bg-brand-500"></div>
                        <span className="action-value text-sm font-medium text-slate-800">
                            {actionLabels[workflowState.action_taken] || workflowState.action_taken}
                        </span>
                    </div>
                </div>
            )}

            <Modal
                isOpen={actionModalOpen}
                onClose={() => setActionModalOpen(false)}
                title={`Confirm Action: ${selectedAction.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}`}
            >
                <div className="workflow-action-modal">
                    <p className="action-description mb-4">
                        You are about to perform "{selectedAction.replace(/_/g, ' ')}" on this leave request.
                    </p>

                    {selectedAction === 'convert_leave_type' && (
                        <div className="form-group mb-4">
                            <label className="block text-sm font-medium text-slate-700 mb-1">Convert To</label>
                            <div className="relative">
                                <select
                                    value={conversionType}
                                    onChange={(e) => setConversionType(e.target.value)}
                                    className="w-full appearance-none h-10 text-sm text-slate-900 bg-white border border-slate-300 rounded-lg shadow-sm focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500 transition-all py-2 pl-3 pr-10"
                                    style={{ color: '#0f172a', backgroundColor: 'white', colorScheme: 'light' }}
                                >
                                    <option value="unpaid" style={{ color: '#0f172a', backgroundColor: 'white' }}>Unpaid Leave</option>
                                    <option value="emergency" style={{ color: '#0f172a', backgroundColor: 'white' }}>Emergency Leave</option>
                                    <option value="annual" style={{ color: '#0f172a', backgroundColor: 'white' }}>Annual Leave</option>
                                    <option value="sick" style={{ color: '#0f172a', backgroundColor: 'white' }}>Sick Leave</option>
                                </select>
                                <ChevronDown className="absolute right-3 top-3 h-4 w-4 text-slate-400 pointer-events-none" />
                            </div>
                        </div>
                    )}

                    <div className="form-group">
                        <label className="block text-sm font-medium text-slate-700 mb-1">
                            {selectedAction === 'convert_leave_type' ? 'Reason (Required)' : 'Comment (Optional)'}
                        </label>
                        <textarea
                            value={actionComment}
                            onChange={e => setActionComment(e.target.value)}
                            placeholder={selectedAction === 'convert_leave_type' ? "State reason for conversion..." : "Add a comment..."}
                            className="action-comment-input w-full border border-slate-300 rounded-lg text-sm text-slate-900 bg-white p-3 shadow-sm focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500 transition-all"
                            style={{ color: '#0f172a', backgroundColor: 'white', colorScheme: 'light' }}
                            rows={3}
                        />
                    </div>
                    <div className="action-modal-buttons mt-6">
                        <Button variant="ghost" onClick={() => setActionModalOpen(false)}>
                            Cancel
                        </Button>
                        <Button
                            onClick={handleAction}
                            isLoading={actionLoading}
                            variant={selectedAction.includes('reject') || selectedAction.includes('not') ? 'danger' : 'primary'}
                            className={cn(
                                'shadow-sm',
                                selectedAction.includes('reject') && 'bg-red-600 hover:bg-red-700',
                                selectedAction === 'convert_leave_type' && 'bg-slate-800 hover:bg-slate-900',
                                selectedAction === 'escalate' && 'bg-purple-600 hover:bg-purple-700',
                                selectedAction === 'request_docs' && 'bg-amber-600 hover:bg-amber-700'
                            )}
                        >
                            {selectedAction.includes('reject') && <AlertTriangle className="w-4 h-4 mr-1.5" />}
                            Confirm Action
                        </Button>
                    </div>
                </div>
            </Modal>
        </div>
    );
};

export default WorkflowStateDisplay;
