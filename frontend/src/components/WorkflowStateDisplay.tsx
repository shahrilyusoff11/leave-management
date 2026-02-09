import React, { useState, useEffect, useCallback } from 'react';
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
    const { showToast } = useToast();
    const [workflowState, setWorkflowState] = useState<LeaveRequestWorkflowState | null>(null);
    const [loading, setLoading] = useState(true);
    const [actionLoading, setActionLoading] = useState(false);
    const [actionModalOpen, setActionModalOpen] = useState(false);
    const [selectedAction, setSelectedAction] = useState<string>('');
    const [actionComment, setActionComment] = useState('');

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

        setActionLoading(true);
        try {
            await api.post(`/leave-requests/${requestId}/workflow-action`, {
                action: selectedAction,
                comment: actionComment
            });
            showToast(`Action "${selectedAction}" completed successfully`, 'success');
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
                    <p className="action-description">
                        You are about to perform "{selectedAction.replace(/_/g, ' ')}" on this leave request.
                    </p>
                    <div className="form-group">
                        <label>Comment (optional)</label>
                        <textarea
                            value={actionComment}
                            onChange={e => setActionComment(e.target.value)}
                            placeholder="Add a comment..."
                            className="action-comment-input"
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
