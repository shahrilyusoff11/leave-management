import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../context/AuthContext';
import api from '../services/api';
import type { LeaveRequestWorkflowState } from '../types';
import { Badge } from './ui/Badge';
import { Button } from './ui/Button';
import { Modal } from './ui/Modal';
import { useToast } from './ui/Toast';
import './WorkflowState.css';

interface WorkflowStateDisplayProps {
    requestId: string;
    currentStatus: string;
    onActionComplete?: () => void;
    showActions?: boolean;
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
    currentStatus,
    onActionComplete,
    showActions = true
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

    const getAvailableActions = (): string[] => {
        if (!workflowState?.current_step) return [];

        const actionType = workflowState.current_step.action_type;
        switch (actionType) {
            case 'approve':
                return ['approve', 'reject'];
            case 'verify':
                return ['verify', 'not_verify', 'request_docs'];
            case 'review':
                return ['approve', 'reject', 'escalate'];
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

            {currentStep && !isComplete && (
                <div className="workflow-current-step">
                    <div className="step-info">
                        <span className="step-label">{currentStep.step_label || currentStep.step_name}</span>
                        <span className="step-role">Awaiting: {currentStep.responsible_role}</span>
                    </div>

                    {showActions && currentStatus === 'pending' && (
                        <div className="workflow-actions">
                            {getAvailableActions().map(action => (
                                <Button
                                    key={action}
                                    size="sm"
                                    variant={action.includes('reject') || action.includes('not') ? 'danger' : 'primary'}
                                    onClick={() => openActionModal(action)}
                                    className="workflow-action-btn"
                                >
                                    {action.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
                                </Button>
                            ))}
                        </div>
                    )}
                    {isHR && !isComplete && (
                        <div className="mt-4 pt-4 border-t border-slate-100 flex justify-end">
                            <Button
                                size="sm"
                                variant="outline"
                                onClick={() => openActionModal('convert_leave_type')}
                                className="text-slate-600 border-slate-300 hover:bg-slate-50"
                            >
                                Convert Leave Type
                            </Button>
                        </div>
                    )}
                </div>
            )}

            {workflowState.action_taken && workflowState.action_taken !== 'pending' && (
                <div className="workflow-last-action">
                    <span className="action-label">Last Action:</span>
                    <span className="action-value">{actionLabels[workflowState.action_taken] || workflowState.action_taken}</span>
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
                            <select
                                value={conversionType}
                                onChange={(e) => setConversionType(e.target.value)}
                                className="w-full text-sm text-slate-900 bg-white border border-slate-300 rounded-md shadow-sm focus:ring-brand-500 focus:border-brand-500 py-2 px-3"
                            >
                                <option value="unpaid" className="text-slate-900">Unpaid Leave</option>
                                <option value="emergency" className="text-slate-900">Emergency Leave</option>
                                <option value="annual" className="text-slate-900">Annual Leave</option>
                                <option value="sick" className="text-slate-900">Sick Leave</option>
                            </select>
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
                            className="action-comment-input w-full border border-slate-300 rounded-md text-sm text-slate-900 bg-white p-2 shadow-sm focus:ring-brand-500 focus:border-brand-500"
                            rows={3}
                        />
                    </div>
                    <div className="action-modal-buttons">
                        <Button variant="ghost" onClick={() => setActionModalOpen(false)}>
                            Cancel
                        </Button>
                        <Button
                            onClick={handleAction}
                            isLoading={actionLoading}
                            variant={selectedAction.includes('reject') || selectedAction.includes('not') ? 'danger' : 'primary'}
                        >
                            Confirm
                        </Button>
                    </div>
                </div>
            </Modal>
        </div>
    );
};

export default WorkflowStateDisplay;
