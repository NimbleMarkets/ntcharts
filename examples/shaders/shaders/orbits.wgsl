fn scene(p: vec3<f32>) -> f32 {
 let t=uni.time*uni.speed;
 let q=vec3<f32>(rotate(p.xz,t*0.4).x,p.y,rotate(p.xz,t*0.4).y);
 let ring=length(vec2<f32>(length(q.xy)-0.85,q.z))-0.18;
 let orb=length(p-vec3<f32>(sin(t)*1.0,cos(t*0.8)*0.65,sin(t*0.6)*0.7))-0.35;
 let k=0.12+uni.detail*0.15;
 let h=clamp(0.5+0.5*(orb-ring)/k,0.0,1.0);
 return mix(orb,ring,h)-k*h*(1.0-h);
}
fn shade(uv: vec2<f32>) -> vec3<f32> {
 let ro=vec3<f32>(0.0,0.3,3.5);
 let rd=normalize(vec3<f32>(uv.x/uni.scale,-uv.y/uni.scale-0.12,-2.0));
 var distance=0.0; var hit=false; var floorHit=false;
 for(var i=0;i<80;i=i+1) {
  let p=ro+rd*distance;
  let object=scene(p); let floorDist=p.y+1.15;
  let stepDist=min(object,floorDist);
  if(stepDist<0.0015) { hit=true; floorHit=floorDist<object; break; }
  distance=distance+stepDist;
  if(distance>16.0) { break; }
 }
 let bg=mix(vec3<f32>(0.02,0.025,0.08),vec3<f32>(0.14,0.07,0.22),clamp(0.5-uv.y*0.3,0.0,1.0));
 if(!hit) { return bg; }
 let p=ro+rd*distance;
 var n=vec3<f32>(0.0,1.0,0.0);
 if(!floorHit) {
  let e=0.002;
  n=normalize(vec3<f32>(scene(p+vec3<f32>(e,0.0,0.0))-scene(p-vec3<f32>(e,0.0,0.0)),scene(p+vec3<f32>(0.0,e,0.0))-scene(p-vec3<f32>(0.0,e,0.0)),scene(p+vec3<f32>(0.0,0.0,e))-scene(p-vec3<f32>(0.0,0.0,e))));
 }
 let light=normalize(vec3<f32>(-0.7,1.5,1.8));
 let diffuse=max(dot(n,light),0.0);
 let spec=pow(max(dot(reflect(-light,n),-rd),0.0),48.0);
 var base=palette(n.y*0.3+n.x*0.2+uni.time*0.03);
 if(floorHit) {
  let checker=abs((floor(p.x*2.0)+floor(p.z*2.0))%2.0);
  base=mix(vec3<f32>(0.04,0.05,0.09),vec3<f32>(0.18,0.2,0.28),checker);
 }
 let fresnel=pow(1.0-max(dot(n,-rd),0.0),3.0);
 let lit=base*(0.2+diffuse*0.8)+vec3<f32>(spec*0.9)+palette(n.x+0.2)*fresnel*0.4;
 return mix(lit,bg,1.0-exp(-distance*distance*0.008));
}
