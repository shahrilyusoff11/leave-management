import React, { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { Lock, Mail } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import api from '../services/api';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';


const loginSchema = z.object({
    email: z.string().email('Invalid email address'),
    password: z.string().min(1, 'Password is required'),
});

type LoginFormData = z.infer<typeof loginSchema>;

const Login: React.FC = () => {
    const { login } = useAuth();
    const navigate = useNavigate();
    const [error, setError] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(false);

    const { register, handleSubmit, formState: { errors } } = useForm<LoginFormData>({
        resolver: zodResolver(loginSchema),
    });

    const onSubmit = async (data: LoginFormData) => {
        setIsLoading(true);
        setError(null);
        try {
            const response = await api.post('/login', data);
            login(response.data.token, response.data.user);
            navigate('/dashboard');
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to login');
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <div className="min-h-screen grid lg:grid-cols-2 bg-white">
            {/* Left: Brand & Pattern */}
            <div className="hidden lg:relative lg:flex lg:flex-col lg:items-center lg:justify-center bg-brand-50 overflow-hidden relative">

                {/* Background Pattern */}
                <div className="absolute inset-0">
                    <img
                        src="/src/assets/login-pattern.svg"
                        alt="Pattern"
                        className="w-full h-full object-cover opacity-60"
                    />
                    <div className="absolute inset-0 bg-gradient-to-br from-white/40 via-transparent to-brand-100/20" />
                </div>

                <div className="relative z-10 px-12 text-center max-w-lg">
                    <motion.div
                        initial={{ scale: 0.9, opacity: 0 }}
                        animate={{ scale: 1, opacity: 1 }}
                        transition={{ duration: 0.5 }}
                        className="mb-8 inline-flex items-center justify-center w-24 h-24 rounded-3xl bg-white shadow-xl shadow-brand-100 border border-brand-50"
                    >
                        <div className="w-12 h-12 rounded-xl bg-brand-600 flex items-center justify-center">
                            <Lock className="w-6 h-6 text-white" />
                        </div>
                    </motion.div>

                    <motion.h1
                        initial={{ y: 20, opacity: 0 }}
                        animate={{ y: 0, opacity: 1 }}
                        transition={{ delay: 0.2 }}
                        className="text-4xl font-bold text-slate-900 mb-6 tracking-tight"
                    >
                        Oh
                    </motion.h1>

                    <motion.p
                        initial={{ y: 20, opacity: 0 }}
                        animate={{ y: 0, opacity: 1 }}
                        transition={{ delay: 0.3 }}
                        className="text-slate-500 text-lg leading-relaxed"
                    >
                        Efficiently manage employee requests, track time-off, and streamline HR workflows in one unified platform.
                    </motion.p>
                </div>
            </div>

            {/* Right: Login Form */}
            <div className="flex items-center justify-center p-8 sm:p-12 md:p-16 lg:p-24 bg-white">
                <div className="w-full max-w-sm space-y-10">
                    <div className="text-center lg:text-left">
                        <h2 className="text-3xl font-semibold tracking-tight text-slate-900">
                            Welcome back
                        </h2>
                        <p className="mt-3 text-slate-500">
                            Please enter your details to sign in
                        </p>
                    </div>

                    <form onSubmit={handleSubmit(onSubmit)} className="space-y-8">
                        {error && (
                            <motion.div
                                initial={{ opacity: 0, height: 0 }}
                                animate={{ opacity: 1, height: 'auto' }}
                                className="p-4 rounded-lg bg-red-50 text-red-600 text-sm border border-red-100 flex items-center gap-3"
                            >
                                <div className="h-2 w-2 rounded-full bg-red-600 flex-shrink-0" />
                                {error}
                            </motion.div>
                        )}

                        <div className="space-y-6">
                            <div className="space-y-2">
                                <label className="text-sm font-medium text-slate-700">
                                    Email Address
                                </label>
                                <div className="relative group">
                                    <Mail className="absolute left-3.5 top-3.5 h-5 w-5 text-slate-400 group-focus-within:text-brand-600 transition-colors" />
                                    <Input
                                        placeholder="name@company.com"
                                        className="pl-11 h-12 bg-white border-slate-200 focus:border-brand-500 focus:ring-4 focus:ring-brand-500/10 transition-all duration-200 rounded-xl"
                                        error={errors.email?.message}
                                        {...register('email')}
                                    />
                                </div>
                            </div>

                            <div className="space-y-2">
                                <div className="flex items-center justify-between">
                                    <label className="text-sm font-medium text-slate-700">
                                        Password
                                    </label>
                                </div>
                                <div className="relative group">
                                    <Lock className="absolute left-3.5 top-3.5 h-5 w-5 text-slate-400 group-focus-within:text-brand-600 transition-colors" />
                                    <Input
                                        type="password"
                                        placeholder="••••••••"
                                        className="pl-11 h-12 bg-white border-slate-200 focus:border-brand-500 focus:ring-4 focus:ring-brand-500/10 transition-all duration-200 rounded-xl"
                                        error={errors.password?.message}
                                        {...register('password')}
                                    />
                                </div>
                                <div className="flex justify-end">
                                    <a href="#" className="text-sm font-medium text-brand-600 hover:text-brand-700 hover:underline">
                                        Forgot password?
                                    </a>
                                </div>
                            </div>
                        </div>

                        <Button
                            type="submit"
                            className="w-full h-12 text-base font-semibold shadow-xl shadow-brand-500/20 hover:shadow-brand-500/30 transition-all duration-300 rounded-xl"
                            size="lg"
                            isLoading={isLoading}
                        >
                            Sign In
                        </Button>
                    </form>

                    <p className="text-center text-sm text-slate-500">
                        Don't have an account?{' '}
                        <a href="#" className="font-medium text-brand-600 hover:text-brand-700 hover:underline">
                            Contact Admin
                        </a>
                    </p>
                </div>
            </div>
        </div>
    );
};

export default Login;
