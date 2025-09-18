package com.example.web.config;

import com.example.core.repository.InMemoryUserRepository;
import com.example.core.repository.UserRepository;
import com.example.core.service.UserService;
import com.example.service.EmailService;
import com.example.service.UserRegistrationService;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.ComponentScan;
import org.springframework.context.annotation.Configuration;

@Configuration
@ComponentScan(basePackages = {
    "com.example.core",
    "com.example.service",
    "com.example.web"
})
public class AppConfig {
    
    @Bean
    public UserRepository userRepository() {
        return new InMemoryUserRepository();
    }
    
    @Bean
    public UserService userService(UserRepository userRepository) {
        return new UserService(userRepository);
    }
    
    @Bean
    public EmailService emailService() {
        return new EmailService();
    }
    
    @Bean
    public UserRegistrationService userRegistrationService(
            UserService userService, 
            EmailService emailService) {
        return new UserRegistrationService(userService, emailService);
    }
}