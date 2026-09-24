fn shade(uv: vec2<f32>) -> vec3<f32> {
 let t=uni.time*uni.speed; let p=rotate(uv,t*0.12)*uni.scale;
 let r=max(length(p),0.025); let a=atan2(p.y,p.x);
 let depth=1.4/r+t*1.4+uni.detail*sin(a*5.0+t)*0.3;
 let bands=pow(0.5+0.5*cos(depth*6.0),18.0);
 let spokes=pow(0.5+0.5*cos(a*12.0+depth*0.4),24.0);
 let glow=(bands+spokes)*0.9+0.09;
 return palette(depth*0.1+a*0.15)*glow*smoothstep(0.04,0.7,r);
}
