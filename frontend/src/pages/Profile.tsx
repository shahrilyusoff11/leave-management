import React, { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { Lock, Mail, Building, Calendar as CalendarIcon, Shield, X, Check } from 'lucide-react';
import api from '../services/api';
import { Card } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { format } from 'date-fns';
import type { User as UserType } from '../types';

const Profile: React.FC = () => {
    const [user, setUser] = useState<UserType | null>(null);
    const [loading, setLoading] = useState(true);
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');
    const [isEditing, setIsEditing] = useState(false);

    // Form for password change
    const {
        register: registerPassword,
        handleSubmit: handleSubmitPassword,
        formState: { errors: passwordErrors, isSubmitting: isPasswordSubmitting },
        reset: resetPassword
    } = useForm();

    // Form for profile update
    const {
        register: registerProfile,
        handleSubmit: handleSubmitProfile,
        formState: { errors: profileErrors, isSubmitting: isProfileSubmitting },
        reset: resetProfile,
        setValue
    } = useForm();

    useEffect(() => {
        const fetchProfile = async () => {
            try {
                const response = await api.get('/profile');
                setUser(response.data);
                // Set initial values for profile form
                setValue('first_name', response.data.first_name);
                setValue('last_name', response.data.last_name);
            } catch (error) {
                console.error("Failed to fetch profile", error);
            } finally {
                setLoading(false);
            }
        };
        fetchProfile();
    }, [setValue]);

    const onUpdateProfile = async (data: any) => {
        try {
            const response = await api.put('/profile', {
                first_name: data.first_name,
                last_name: data.last_name
            });
            setUser(response.data);
            setIsEditing(false);
            setMessage("Profile updated successfully");
            setTimeout(() => setMessage(''), 3000);
        } catch (err: any) {
            setError(err.response?.data?.error || "Failed to update profile");
            setTimeout(() => setError(''), 3000);
        }
    };

    const onChangePassword = async (data: any) => {
        setMessage('');
        setError('');

        if (data.new_password !== data.confirm_password) {
            setError("Passwords do not match");
            return;
        }

        try {
            await api.put('/change-password', {
                current_password: data.old_password,
                new_password: data.new_password
            });
            setMessage("Password changed successfully");
            resetPassword();
        } catch (err: any) {
            setError(err.response?.data?.error || "Failed to change password");
        }
    };

    if (loading) return <div className="p-8 text-center text-slate-500">Loading profile...</div>;
    if (!user) return <div className="p-8 text-center text-slate-500">User not found</div>;

    return (
        <div className="space-y-6 max-w-4xl mx-auto">
            <div>
                <h1 className="text-2xl font-bold text-slate-900">My Profile</h1>
                <p className="text-slate-500 mt-1">Manage your account settings</p>
            </div>

            {message && <div className="p-4 bg-green-50 text-green-700 rounded-lg border border-green-200">{message}</div>}
            {error && <div className="p-4 bg-red-50 text-red-700 rounded-lg border border-red-200">{error}</div>}

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="md:col-span-1">
                    <Card className="p-6">
                        <div className="text-center mb-6">
                            <div className="w-24 h-24 bg-brand-100 rounded-full flex items-center justify-center mx-auto mb-4">
                                <span className="text-3xl font-bold text-brand-600">
                                    {user.first_name[0]}{user.last_name[0]}
                                </span>
                            </div>
                            {!isEditing ? (
                                <>
                                    <h2 className="text-xl font-bold text-slate-900">{user.first_name} {user.last_name}</h2>
                                    <p className="text-slate-500 capitalize">{user.role}</p>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        className="mt-4"
                                        onClick={() => setIsEditing(true)}
                                    >
                                        Edit Profile
                                    </Button>
                                </>
                            ) : (
                                <form onSubmit={handleSubmitProfile(onUpdateProfile)} className="space-y-3 mt-4">
                                    <div>
                                        <Input
                                            {...registerProfile('first_name', { required: 'Required' })}
                                            placeholder="First Name"
                                            className="text-center"
                                            error={profileErrors.first_name?.message as string}
                                        />
                                    </div>
                                    <div>
                                        <Input
                                            {...registerProfile('last_name', { required: 'Required' })}
                                            placeholder="Last Name"
                                            className="text-center"
                                            error={profileErrors.last_name?.message as string}
                                        />
                                    </div>
                                    <div className="flex gap-2 justify-center">
                                        <Button
                                            type="button"
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => {
                                                setIsEditing(false);
                                                resetProfile({
                                                    first_name: user.first_name,
                                                    last_name: user.last_name
                                                });
                                            }}
                                        >
                                            <X className="h-4 w-4" />
                                        </Button>
                                        <Button
                                            type="submit"
                                            size="sm"
                                            isLoading={isProfileSubmitting}
                                        >
                                            <Check className="h-4 w-4" />
                                        </Button>
                                    </div>
                                </form>
                            )}
                        </div>

                        <div className="mt-6 flex flex-col gap-3 text-sm text-slate-600 px-2 pt-6 border-t border-slate-100">
                            <div className="flex items-center gap-3">
                                <div className="p-2 bg-slate-50 rounded-lg">
                                    <Mail className="h-4 w-4 text-slate-400" />
                                </div>
                                <div className="overflow-hidden">
                                    <p className="text-xs text-slate-400">Email Address</p>
                                    <p className="truncate font-medium text-slate-700">{user.email}</p>
                                </div>
                            </div>
                            <div className="flex items-center gap-3">
                                <div className="p-2 bg-slate-50 rounded-lg">
                                    <Building className="h-4 w-4 text-slate-400" />
                                </div>
                                <div>
                                    <p className="text-xs text-slate-400">Department</p>
                                    <p className="font-medium text-slate-700">{(user as any).department?.name || (user as any).department}</p>
                                </div>
                            </div>
                            <div className="flex items-center gap-3">
                                <div className="p-2 bg-slate-50 rounded-lg">
                                    <Shield className="h-4 w-4 text-slate-400" />
                                </div>
                                <div>
                                    <p className="text-xs text-slate-400">Position</p>
                                    <p className="font-medium text-slate-700">{user.position}</p>
                                </div>
                            </div>
                            <div className="flex items-center gap-3">
                                <div className="p-2 bg-slate-50 rounded-lg">
                                    <CalendarIcon className="h-4 w-4 text-slate-400" />
                                </div>
                                <div>
                                    <p className="text-xs text-slate-400">Joined Date</p>
                                    <p className="font-medium text-slate-700">{format(new Date(user.joined_date), 'MMM yyyy')}</p>
                                </div>
                            </div>
                        </div>
                    </Card>
                </div>

                <div className="md:col-span-2">
                    <Card className="p-6">
                        <h3 className="text-lg font-semibold text-slate-900 mb-6 flex items-center gap-2 pb-4 border-b border-slate-100">
                            <Lock className="h-5 w-5 text-brand-500" />
                            Change Password
                        </h3>

                        <form onSubmit={handleSubmitPassword(onChangePassword)} className="space-y-4 max-w-lg">
                            <div className="space-y-1">
                                <label className="text-sm font-medium text-slate-700">Current Password</label>
                                <Input
                                    type="password"
                                    {...registerPassword('old_password', { required: 'Required' })}
                                    error={passwordErrors.old_password?.message as string}
                                    placeholder="Enter current password"
                                />
                            </div>
                            <div className="space-y-1">
                                <label className="text-sm font-medium text-slate-700">New Password</label>
                                <Input
                                    type="password"
                                    {...registerPassword('new_password', { required: 'Required', minLength: { value: 8, message: 'Min 8 chars' } })}
                                    error={passwordErrors.new_password?.message as string}
                                    placeholder="Enter new password"
                                />
                            </div>
                            <div className="space-y-1">
                                <label className="text-sm font-medium text-slate-700">Confirm New Password</label>
                                <Input
                                    type="password"
                                    {...registerPassword('confirm_password', { required: 'Required' })}
                                    error={passwordErrors.confirm_password?.message as string}
                                    placeholder="Confirm new password"
                                />
                            </div>
                            <div className="flex justify-end pt-4">
                                <Button type="submit" isLoading={isPasswordSubmitting}>Update Password</Button>
                            </div>
                        </form>
                    </Card>
                </div>
            </div>
        </div>
    );
};

export default Profile;

